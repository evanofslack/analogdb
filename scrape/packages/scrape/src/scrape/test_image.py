from io import BytesIO
from typing import Dict, List, Tuple
from unittest.mock import MagicMock, patch

import numpy as np
import pytest
from PIL import Image

from .constants import AWS_BUCKET_PHOTOS, CLOUDFRONT_URL, LOW_RES, USER_AGENT
from .image import (
    ImageProcessor,
    extract_palette,
    image_bytes_hash,
    image_hash,
    oklab_to_srgb,
    srgb_to_oklab,
)
from .models import Color, RedditPost


class FakeS3:
    def __init__(self):
        self.puts: List[Tuple[str, str, bytes, str]] = []

    def put_object(self, bucket: str, key: str, body: bytes, content_type: str) -> None:
        self.puts.append((bucket, key, body, content_type))


def jpeg_bytes(width: int = 1600, height: int = 1200) -> bytes:
    x = np.linspace(0, 255, width)[None, :].repeat(height, axis=0)
    y = np.linspace(0, 255, height)[:, None].repeat(width, axis=1)
    pixels = np.stack([x, y, 255 - x], axis=2).astype(np.uint8)
    buf = BytesIO()
    Image.fromarray(pixels).save(buf, "JPEG")
    return buf.getvalue()


def fake_response(content: bytes, content_type: str) -> MagicMock:
    resp = MagicMock()
    resp.content = content
    resp.headers = {"content-type": content_type}
    return resp


def make_post(permalink: str, subreddit: str = "analog") -> RedditPost:
    return RedditPost(
        image_url=f"https://i.redd.it/{permalink.rsplit('/', 1)[-1]}.jpg",
        subreddit=subreddit,
        title="title",
        selftext=None,
        author="u/author",
        permalink=permalink,
        score=1,
        nsfw=False,
        time=0,
        sprocket=False,
    )


def process(
    post: RedditPost, content: bytes, content_type: str = "image/jpeg"
) -> Tuple[FakeS3, object]:
    s3 = FakeS3()
    with patch(
        "scrape.image.requests.Session.get",
        return_value=fake_response(content, content_type),
    ):
        result = ImageProcessor().process(post, s3)
    return s3, result


def gray_image(noise: int) -> Image.Image:
    rng = np.random.default_rng(1)
    base = np.tile(np.linspace(0, 255, 400), (300, 1))[..., None].repeat(3, axis=2)
    jitter = rng.integers(-noise, noise + 1, size=base.shape)
    pixels = np.clip(base + jitter, 0, 255).astype(np.uint8)
    return Image.fromarray(pixels)


def tinted_image(tint: Tuple[float, float, float]) -> Image.Image:
    base = np.tile(np.linspace(0.2, 1.0, 400), (300, 1))[..., None]
    pixels = (base * np.array(tint)[None, None, :]).astype(np.uint8)
    return Image.fromarray(pixels)


PERMALINK = "https://www.reddit.com/r/analog/comments/abc/title"


class TestProcess:
    def test_urls_are_cloudfront_for_uploaded_keys(self):
        s3, result = process(make_post(PERMALINK), jpeg_bytes())

        keys = {key for _, key, _, _ in s3.puts}
        assert len(result.images) == 4
        for image in result.images:
            key = image.url.removeprefix(f"{CLOUDFRONT_URL}/")
            assert image.url == f"{CLOUDFRONT_URL}/{key}"
            assert key in keys
            assert "amazonaws.com" not in image.url
        assert all(bucket == AWS_BUCKET_PHOTOS for bucket, _, _, _ in s3.puts)

    def test_hash_is_of_uploaded_low_image(self):
        s3, result = process(make_post(PERMALINK), jpeg_bytes())

        low = next(i for i in result.images if i.resolution == "low")
        body = next(b for _, k, b, _ in s3.puts if low.url.endswith(k))
        assert result.image_hash == image_bytes_hash(body)

    def test_keys_are_deterministic(self):
        content = jpeg_bytes()
        first, _ = process(make_post(PERMALINK), content)
        second, _ = process(make_post(PERMALINK), content)
        other, _ = process(make_post(PERMALINK + "2"), content)

        first_keys = [key for _, key, _, _ in first.puts]
        assert len(set(first_keys)) == 4
        assert first_keys == [key for _, key, _, _ in second.puts]
        assert not set(first_keys) & {key for _, key, _, _ in other.puts}

    def test_raw_is_original_bytes(self):
        content = jpeg_bytes()
        s3, result = process(make_post(PERMALINK), content)

        raw = next(i for i in result.images if i.resolution == "raw")
        body = next(b for _, k, b, _ in s3.puts if raw.url.endswith(k))
        assert body == content
        assert (raw.width, raw.height) == (1600, 1200)
        assert (result.width, result.height) == (1600, 1200)

    def test_resized_sizes(self):
        _, result = process(make_post(PERMALINK), jpeg_bytes())

        sizes: Dict[str, Tuple[int, int]] = {
            i.resolution: (i.width, i.height) for i in result.images
        }
        assert sizes == {
            "low": (720, 540),
            "medium": (1080, 810),
            "high": (1440, 1080),
            "raw": (1600, 1200),
        }

    def test_jpg_content_type(self):
        s3, result = process(make_post(PERMALINK), jpeg_bytes(), "image/jpg")

        assert len(result.images) == 4
        assert all(ct == "image/jpg" for _, _, _, ct in s3.puts)
        for _, key, body, _ in s3.puts:
            assert key.endswith(".jpg")
            assert Image.open(BytesIO(body)).format == "JPEG"

    def test_invalid_content_type_raises(self):
        with pytest.raises(ValueError):
            process(make_post(PERMALINK), b"<html></html>", "text/html")

    def test_sends_analogdb_user_agent(self):
        assert ImageProcessor().session.headers["User-Agent"] == USER_AGENT

    def test_colors_are_the_low_res_palette(self):
        _, result = process(make_post(PERMALINK), jpeg_bytes())

        source = Image.open(BytesIO(jpeg_bytes()))
        low, _, _ = ImageProcessor().resize_image(source, LOW_RES)
        assert result.colors == [Color(h, p) for h, p in extract_palette(low)]
        assert 1 < len(result.colors) <= 5

    def test_download_colors(self):
        buf = BytesIO()
        Image.new("RGB", (300, 200), (200, 30, 40)).save(buf, "PNG")
        with patch(
            "scrape.image.requests.Session.get",
            return_value=fake_response(buf.getvalue(), "image/png"),
        ) as get:
            colors = ImageProcessor().download_colors("https://i.analogdb.com/a.png")

        assert colors == [Color("#c81e28", 1.0)]
        get.assert_called_once()


def hash_distance(a: str, b: str) -> int:
    return bin(int(a, 16) ^ int(b, 16)).count("1")


def photo(seed: int) -> Image.Image:
    """A smooth random image, so its hash has detail to keep"""
    rng = np.random.default_rng(seed)
    small = rng.integers(0, 256, size=(12, 16, 3)).astype(np.uint8)
    return Image.fromarray(small).resize((800, 600), Image.Resampling.BICUBIC)


def reencode(image: Image.Image, quality: int, size=None) -> Image.Image:
    if size:
        image = image.resize(size, Image.Resampling.LANCZOS)
    buf = BytesIO()
    image.save(buf, "JPEG", quality=quality)
    buf.seek(0)
    return Image.open(buf)


class TestImageHash:
    def test_is_16_hex_characters(self):
        h = image_hash(photo(1))
        assert len(h) == 16
        int(h, 16)

    def test_same_image_same_hash(self):
        assert image_hash(photo(1)) == image_hash(photo(1))

    def test_reencoded_copy_is_close(self):
        original = photo(1)
        for quality, size in [(95, None), (70, None), (85, (400, 300))]:
            copy = reencode(original, quality, size)
            assert hash_distance(image_hash(original), image_hash(copy)) <= 4

    def test_different_images_are_far(self):
        hashes = [image_hash(photo(seed)) for seed in range(5)]
        for i, a in enumerate(hashes):
            for b in hashes[i + 1 :]:
                assert hash_distance(a, b) > 10

    def test_bytes_hash_matches_image_hash(self):
        buf = BytesIO()
        photo(2).save(buf, "PNG")
        assert image_bytes_hash(buf.getvalue()) == image_hash(photo(2))


class TestGrayscale:
    def test_noisy_gray_is_grayscale(self):
        assert ImageProcessor().is_grayscale(gray_image(noise=2))

    def test_muted_color_is_not_grayscale(self):
        assert not ImageProcessor().is_grayscale(tinted_image((150, 170, 190)))

    def test_sepia_is_not_grayscale(self):
        assert not ImageProcessor().is_grayscale(tinted_image((240, 200, 150)))

    def test_toned_bw_subreddit_post_is_color(self):
        buf = BytesIO()
        tinted_image((240, 200, 150)).save(buf, "JPEG")
        post = make_post(PERMALINK, subreddit="analog_bw")

        _, result = process(post, buf.getvalue())
        assert not result.grayscale


def split_image(share: float, a, b, width: int = 200, height: int = 100):
    """Pixels of b scattered over a, so the border crop keeps the whole frame"""
    rng = np.random.default_rng(3)
    mask = np.zeros(width * height, dtype=bool)
    mask[rng.permutation(width * height)[: round(width * height * (1 - share))]] = True
    mask[0] = False
    pixels = np.where(mask[:, None], np.array(b), np.array(a)).astype(np.uint8)
    return Image.fromarray(pixels.reshape(height, width, 3))


class TestPalette:
    def test_oklab_round_trip(self):
        rgb = np.random.default_rng(0).integers(0, 256, size=(500, 3))
        assert np.array_equal(oklab_to_srgb(srgb_to_oklab(rgb)), rgb)

    def test_oklab_white_and_black(self):
        lab = srgb_to_oklab(np.array([[255, 255, 255], [0, 0, 0]]))
        assert np.allclose(lab, [[1, 0, 0], [0, 0, 0]], atol=1e-6)

    def test_solid_color(self):
        palette = extract_palette(Image.new("RGB", (300, 200), (200, 30, 40)))
        assert palette == [("#c81e28", 1.0)]

    def test_split_shares(self):
        image = split_image(0.7, (30, 90, 200), (240, 200, 40))
        palette = dict(extract_palette(image))
        assert set(palette) == {"#1e5ac8", "#f0c828"}
        assert palette["#1e5ac8"] == pytest.approx(0.7, abs=0.01)
        assert palette["#f0c828"] == pytest.approx(0.3, abs=0.01)

    def test_close_colors_merge(self):
        image = split_image(0.5, (120, 120, 120), (122, 121, 120))
        palette = extract_palette(image)
        assert len(palette) == 1
        assert palette[0][1] == pytest.approx(1.0)

    def test_many_colors(self):
        palette = extract_palette(photo(4))
        assert 1 < len(palette) <= 5
        shares = [p for _, p in palette]
        assert sum(shares) <= 1.0
        assert shares == sorted(shares, reverse=True)
        assert all(len(h) == 7 and h.startswith("#") for h, _ in palette)

    def test_shares_are_over_the_whole_image(self):
        stripes = [
            (220, 30, 30),
            (30, 180, 40),
            (30, 60, 220),
            (240, 220, 40),
            (40, 220, 230),
            (210, 40, 200),
            (10, 10, 10),
            (240, 140, 20),
        ]
        pixels = np.repeat(np.array(stripes, dtype=np.uint8), 10, axis=0)
        pixels = np.repeat(pixels[:, None, :], 100, axis=1)
        pixels[0, 0] = (255, 255, 255)
        palette = extract_palette(Image.fromarray(pixels))
        assert len(palette) == 5
        for _, share in palette:
            assert share == pytest.approx(1 / 8, abs=0.01)
        assert sum(p for _, p in palette) == pytest.approx(5 / 8, abs=0.01)

    def test_large_image_is_sampled(self):
        image = split_image(0.6, (20, 140, 60), (200, 60, 160), 720, 540)
        palette = dict(extract_palette(image))
        assert palette["#148c3c"] == pytest.approx(0.6, abs=0.01)

    def test_deterministic(self):
        image = photo(5)
        assert extract_palette(image) == extract_palette(image.copy())

    def test_tiny_images(self):
        assert extract_palette(Image.new("RGB", (1, 1), (9, 9, 9))) == [
            ("#090909", 1.0)
        ]
        two = Image.new("RGB", (2, 1), (0, 0, 0))
        two.putpixel((1, 0), (255, 255, 255))
        assert len(extract_palette(two)) <= 2

    def test_other_modes(self):
        image = Image.new("L", (40, 40), 128)
        assert extract_palette(image) == [("#808080", 1.0)]
        rgba = Image.new("RGBA", (40, 40), (10, 200, 30, 255))
        assert extract_palette(rgba) == [("#0ac81e", 1.0)]

    def test_processor_method(self):
        image = photo(6)
        assert ImageProcessor().extract_palette(image) == extract_palette(image)

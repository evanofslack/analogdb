from io import BytesIO
from typing import Dict, List, Tuple
from unittest.mock import MagicMock, patch

import numpy as np
import pytest
from PIL import Image

from .constants import AWS_BUCKET_PHOTOS, CLOUDFRONT_URL
from .image import ImageProcessor
from .models import RedditPost


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
        "scrape.image.requests.get",
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

    def test_colors_extracted(self):
        _, result = process(make_post(PERMALINK), jpeg_bytes())
        assert result.colors


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

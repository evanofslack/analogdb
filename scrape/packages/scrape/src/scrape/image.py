import hashlib
import warnings
from io import BytesIO
from typing import List, Optional, Tuple

import numpy as np
import requests
from PIL import Image, ImageChops
from retry import retry
from scipy.cluster.vq import kmeans2, vq

from .constants import (
    AWS_BUCKET_PHOTOS,
    CLOUDFRONT_URL,
    COLOR_LIMIT,
    GRAYSCALE_SUBREDDITS,
    GRAYSCALE_TOLERANCE,
    HIGH_RES,
    IMAGE_TIMEOUT,
    JPEG_QUALITY,
    LOW_RES,
    MEDIUM_RES,
    PALETTE_CLUSTERS,
    PALETTE_MERGE_DISTANCE,
    PALETTE_SAMPLE,
    PALETTE_SEED,
    USER_AGENT,
    VALID_CONTENT,
)
from .models import Color, PostImages, RedditPost, S3Image
from .s3 import S3


def image_hash(image: Image.Image) -> str:
    """64 bit dHash as 16 hex characters. Each bit says whether a pixel of the
    9x8 grayscale thumbnail is brighter than its left neighbour, so a resized or
    re-encoded copy keeps nearly every bit."""
    small = image.convert("L").resize((9, 8), Image.Resampling.LANCZOS)
    pixels = np.asarray(small, dtype=np.int16)
    bits = (pixels[:, 1:] > pixels[:, :-1]).flatten()
    value = 0
    for bit in bits:
        value = (value << 1) | int(bit)
    return f"{value:016x}"


def image_bytes_hash(content: bytes) -> str:
    with Image.open(BytesIO(content)) as image:
        return image_hash(image)


def channel_spread(image: Image.Image) -> np.ndarray:
    a = np.asarray(image.convert("RGB"), dtype=np.int16)
    return a.max(axis=2) - a.min(axis=2)


_LINEAR_TO_LMS = np.array(
    [
        [0.4122214708, 0.5363325363, 0.0514459929],
        [0.2119034982, 0.6806995451, 0.1073969566],
        [0.0883024619, 0.2817188376, 0.6299787005],
    ]
)

_LMS_TO_OKLAB = np.array(
    [
        [0.2104542553, 0.7936177850, -0.0040720468],
        [1.9779984951, -2.4285922050, 0.4505937099],
        [0.0259040371, 0.7827717662, -0.8086757660],
    ]
)

_OKLAB_TO_LMS = np.array(
    [
        [1.0, 0.3963377774, 0.2158037573],
        [1.0, -0.1055613458, -0.0638541728],
        [1.0, -0.0894841775, -1.2914855480],
    ]
)

_LMS_TO_LINEAR = np.array(
    [
        [4.0767416621, -3.3077115913, 0.2309699292],
        [-1.2684380046, 2.6097574011, -0.3413193965],
        [-0.0041960863, -0.7034186147, 1.7076147010],
    ]
)


def _transform(rows: np.ndarray, matrix: np.ndarray) -> np.ndarray:
    return np.einsum("nj,ij->ni", rows, matrix)


def srgb_to_oklab(rgb: np.ndarray) -> np.ndarray:
    """Rows of 0-255 sRGB to rows of OKLab"""
    c = np.asarray(rgb, dtype=np.float64) / 255.0
    linear = np.where(c <= 0.04045, c / 12.92, ((c + 0.055) / 1.055) ** 2.4)
    lms = np.cbrt(_transform(linear, _LINEAR_TO_LMS))
    return _transform(lms, _LMS_TO_OKLAB)


def oklab_to_srgb(lab: np.ndarray) -> np.ndarray:
    """Rows of OKLab to rows of 0-255 sRGB, clipped to the gamut"""
    lms = _transform(np.asarray(lab, dtype=np.float64), _OKLAB_TO_LMS) ** 3
    linear = np.clip(_transform(lms, _LMS_TO_LINEAR), 0.0, 1.0)
    c = np.where(
        linear <= 0.0031308, linear * 12.92, 1.055 * linear ** (1 / 2.4) - 0.055
    )
    return np.clip(np.round(c * 255.0), 0, 255).astype(np.uint8)


def resize_image(
    image: Image.Image, size: Optional[Tuple[int, int]]
) -> Tuple[Image.Image, int, int]:
    if not size:
        return image, image.width, image.height
    img_resized = image.copy()
    img_resized.thumbnail(size, Image.Resampling.LANCZOS)
    return img_resized, img_resized.width, img_resized.height


def remove_border(image: Image.Image) -> Image.Image:
    bg = Image.new(image.mode, image.size, image.getpixel((0, 0)))
    diff = ImageChops.difference(image, bg)
    diff = ImageChops.add(diff, diff, 1.0, -100)
    bbox = diff.getbbox()
    return image.crop(bbox) if bbox else image


def prepare_image_for_analysis(image: Image.Image) -> Image.Image:
    resized, _, _ = resize_image(image, LOW_RES)
    return remove_border(resized)


def _fit_clusters(lab: np.ndarray) -> np.ndarray:
    rng = np.random.default_rng(PALETTE_SEED)
    sample = lab
    if len(lab) > PALETTE_SAMPLE:
        sample = lab[np.sort(rng.choice(len(lab), PALETTE_SAMPLE, replace=False))]
    k = min(PALETTE_CLUSTERS, len(np.unique(sample, axis=0)))
    if k <= 1:
        return sample.mean(axis=0, keepdims=True)
    with warnings.catch_warnings():
        warnings.simplefilter("ignore")
        centroids, _ = kmeans2(sample, k, minit="++", rng=rng)
    return centroids


def _merge_clusters(
    centroids: np.ndarray, counts: np.ndarray
) -> Tuple[np.ndarray, np.ndarray]:
    centroids, counts = list(centroids), list(counts)
    while len(centroids) > 1:
        points = np.array(centroids)
        dist = np.linalg.norm(points[:, None, :] - points[None, :, :], axis=2)
        dist[np.triu_indices(len(points))] = np.inf
        i, j = np.unravel_index(np.argmin(dist), dist.shape)
        if dist[i, j] >= PALETTE_MERGE_DISTANCE:
            break
        total = counts[i] + counts[j]
        centroids[j] = (centroids[i] * counts[i] + centroids[j] * counts[j]) / total
        counts[j] = total
        del centroids[i], counts[i]
    return np.array(centroids), np.array(counts)


def extract_palette(
    image: Image.Image, limit: int = COLOR_LIMIT
) -> List[Tuple[str, float]]:
    """Dominant colors as (hex, share of the whole image), largest first. Pixels
    are clustered in OKLab so close colors group the way they look, and shares
    are not renormalized over the kept colors."""
    prepared = prepare_image_for_analysis(image.convert("RGB"))
    lab = srgb_to_oklab(np.asarray(prepared).reshape(-1, 3))

    labels, _ = vq(lab, _fit_clusters(lab))
    counts = np.bincount(labels)
    kept = np.flatnonzero(counts)
    centroids = np.array([lab[labels == i].mean(axis=0) for i in kept])
    centroids, counts = _merge_clusters(centroids, counts[kept])

    order = np.argsort(-counts, kind="stable")[:limit]
    rgb = oklab_to_srgb(centroids[order])
    return [
        ("#{:02x}{:02x}{:02x}".format(*map(int, c)), round(float(n) / len(lab), 8))
        for c, n in zip(rgb, counts[order])
    ]


class ImageProcessor:
    _content_suffix = {
        "image/png": "png",
        "image/jpeg": "jpeg",
        "image/jpg": "jpg",
        "image/gif": "gif",
    }

    _content_format = {
        "image/jpeg": "JPEG",
        "image/jpg": "JPEG",
        "image/png": "PNG",
        "image/gif": "GIF",
    }

    def process(self, post: RedditPost, s3: S3) -> PostImages:
        content, content_type = self._download(post.image_url)
        prefix = self._key_prefix(post.permalink)
        sizes = {"low": LOW_RES, "medium": MEDIUM_RES, "high": HIGH_RES}

        with Image.open(BytesIO(content)) as image:
            width, height = image.width, image.height
            resized = {res: self.resize_image(image, dim) for res, dim in sizes.items()}

        images: List[S3Image] = []
        hash = None
        for res, (img, w, h) in resized.items():
            body = self.image_to_bytes(image=img, content_type=content_type)
            url = self._upload_image(s3, prefix, res, body, content_type)
            images.append(S3Image(resolution=res, url=url, width=w, height=h))
            # hash the low res bytes as uploaded, the same bytes a backfill reads
            if res == "low":
                hash = image_bytes_hash(body)

        url = self._upload_image(s3, prefix, "raw", content, content_type)
        images.append(S3Image(resolution="raw", url=url, width=width, height=height))

        low = resized["low"][0]
        return PostImages(
            images=images,
            colors=self.colors(low),
            grayscale=self._grayscale(low, post.subreddit),
            width=width,
            height=height,
            image_hash=hash,
        )

    def is_grayscale(self, image: Image.Image) -> bool:
        spread = channel_spread(image)
        return bool(np.percentile(spread, 99) <= GRAYSCALE_TOLERANCE)

    def _grayscale(self, image: Image.Image, subreddit: str) -> bool:
        return subreddit in GRAYSCALE_SUBREDDITS or self.is_grayscale(image)

    def resize_image(
        self, image: Image.Image, size: Optional[Tuple[int, int]]
    ) -> Tuple[Image.Image, int, int]:
        return resize_image(image, size)

    def extract_palette(self, image: Image.Image) -> List[Tuple[str, float]]:
        return extract_palette(image)

    def colors(self, image: Image.Image) -> List[Color]:
        return [Color(hex=h, percent=p) for h, p in extract_palette(image)]

    def download_colors(self, url: str) -> List[Color]:
        content, _ = self._download(url)
        with Image.open(BytesIO(content)) as image:
            return self.colors(image)

    def image_to_bytes(self, image: Image.Image, content_type: str) -> bytes:
        image_bytes = BytesIO()
        format_name = self._content_format[content_type]
        if format_name == "JPEG":
            image.save(image_bytes, format_name, quality=JPEG_QUALITY, optimize=True)
        else:
            image.save(image_bytes, format_name)
        return image_bytes.getvalue()

    @retry(delay=1, tries=5)
    def _fetch(self, url: str) -> requests.Response:
        resp = requests.get(
            url, headers={"User-Agent": USER_AGENT}, timeout=IMAGE_TIMEOUT
        )
        resp.raise_for_status()
        return resp

    def _download(self, url: str) -> Tuple[bytes, str]:
        resp = self._fetch(url)
        content_type = resp.headers.get("content-type", "").split(";")[0].strip()
        if content_type not in VALID_CONTENT:
            raise ValueError(f"Invalid content type: {content_type}")
        return resp.content, content_type

    def _key_prefix(self, permalink: str) -> str:
        return hashlib.sha256(permalink.encode()).hexdigest()[:24]

    def _upload_image(
        self, s3: S3, prefix: str, resolution: str, body: bytes, content_type: str
    ) -> str:
        bucket = AWS_BUCKET_PHOTOS
        filename = f"{prefix}-{resolution}.{self._content_suffix[content_type]}"
        s3.put_object(bucket, filename, body, content_type)

        url = f"{CLOUDFRONT_URL}/{filename}"
        return url

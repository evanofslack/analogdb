import hashlib
from io import BytesIO
from typing import Dict, List, Optional, Tuple

import extcolors
import numpy as np
import requests
import webcolors
from PIL import Image, ImageChops
from retry import retry
from scipy.spatial import KDTree

from .constants import (
    AWS_BUCKET_PHOTOS,
    CLOUDFRONT_URL,
    COLOR_LIMIT,
    COLOR_TOLERANCE,
    GRAYSCALE_SUBREDDITS,
    GRAYSCALE_TOLERANCE,
    HIGH_RES,
    IMAGE_TIMEOUT,
    JPEG_QUALITY,
    LOW_RES,
    MEDIUM_RES,
    VALID_CONTENT,
)
from .models import Color, PostImages, RedditPost, S3Image
from .s3 import S3


def channel_spread(image: Image.Image) -> np.ndarray:
    a = np.asarray(image.convert("RGB"), dtype=np.int16)
    return a.max(axis=2) - a.min(axis=2)


class ImageProcessor:
    HTML_OVERRIDES: Dict[str, str] = {
        "silver": "gray",
        "fuschia": "purple",
        "blue": "teal",
        "aqua": "teal",
    }

    CSS_OVERRIDES: Dict[str, str] = {
        "maroon": "red",
        "firebrick": "red",
        "salmon": "red",
        "darkred": "red",
        "lightsalmon": "orange",
        "orange": "orange",
        "darkorange": "orange",
        "orangered": "orange",
        "coral": "orange",
        "mediumseagreen": "green",
        "seagreen": "green",
        "yellowgreen": "green",
        "greenyellow": "green",
        "steelblue": "teal",
        "lightsteelblue": "teal",
        "mediumaquamarine": "teal",
        "darkcyan": "teal",
        "darkseagreen": "teal",
        "paleturquoise": "teal",
        "cadetblue": "teal",
        "cornflowerblue": "teal",
        "lightblue": "teal",
        "skyblue": "teal",
        "lightskyblue": "teal",
        "wienna": "brown",
        "chocolate": "brown",
        "rosybrown": "brown",
        "saddlebrown": "brown",
        "darkkhaki": "brown",
        "darksalmon": "brown",
        "brown": "brown",
        "burlywood": "tan",
        "bisque": "tan",
        "antiquewhite ": "tan",
        "blanchedalmond": "tan",
        "peru": "tan",
        "sandybrown": "tan",
        "papayawhip ": "tan",
        "tan": "tan",
        "navajowhite ": "tan",
        "moccasin  ": "tan",
        "peachpuff": "tan",
        "wheat": "tan",
        "khaki": "tan",
        "darkgray": "gray",
        "dimgray": "gray",
        "thistle": "gray",
        "silver": "gray",
        "lightslategray": "gray",
        "darkslategray": "gray",
        "gainsboro": "gray",
        "lightyellow": "yellow",
        "lightgoldenrodyellow": "yellow",
        "lemonchiffon": "yellow",
        "goldenrod": "yellow",
        "darkolivegreen": "olive",
        "olivedrab": "olive",
        "darkslateblue": "navy",
        "midnightblue": "navy",
        "violet": "purple",
        "lightcoral": "purple",
        "lightpink": "purple",
        "royalblue": "purple",
        "seashell": "white",
        "snow": "white",
    }

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

    def __init__(self):
        self._css_names = list(webcolors.names(spec=webcolors.CSS3))
        self._css_color_tree = self._build_css_color_tree()
        self._html_names = list(webcolors.names(spec=webcolors.HTML4))
        self._html_color_tree = self._build_html_color_tree()

    def process(self, post: RedditPost, s3: S3) -> PostImages:
        content, content_type = self._download(post.image_url)
        prefix = self._key_prefix(post.permalink)
        sizes = {"low": LOW_RES, "medium": MEDIUM_RES, "high": HIGH_RES}

        with Image.open(BytesIO(content)) as image:
            width, height = image.width, image.height
            resized = {res: self.resize_image(image, dim) for res, dim in sizes.items()}

        images: List[S3Image] = []
        for res, (img, w, h) in resized.items():
            body = self.image_to_bytes(image=img, content_type=content_type)
            url = self._upload_image(s3, prefix, res, body, content_type)
            images.append(S3Image(resolution=res, url=url, width=w, height=h))

        url = self._upload_image(s3, prefix, "raw", content, content_type)
        images.append(S3Image(resolution="raw", url=url, width=width, height=height))

        low = resized["low"][0]
        return PostImages(
            images=images,
            colors=self._extract_colors(low),
            grayscale=self._grayscale(low, post.subreddit),
            width=width,
            height=height,
        )

    def is_grayscale(self, image: Image.Image) -> bool:
        spread = channel_spread(image)
        return bool(np.percentile(spread, 99) <= GRAYSCALE_TOLERANCE)

    def _grayscale(self, image: Image.Image, subreddit: str) -> bool:
        return subreddit in GRAYSCALE_SUBREDDITS or self.is_grayscale(image)

    def resize_image(
        self, image: Image.Image, size: Optional[Tuple[int, int]]
    ) -> Tuple[Image.Image, int, int]:
        if not size:
            return image, image.width, image.height
        img_resized = image.copy()
        img_resized.thumbnail(size, Image.Resampling.LANCZOS)
        return img_resized, img_resized.width, img_resized.height

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
        resp = requests.get(url, timeout=IMAGE_TIMEOUT)
        resp.raise_for_status()
        return resp

    def _download(self, url: str) -> Tuple[bytes, str]:
        resp = self._fetch(url)
        content_type = resp.headers.get("content-type", "").split(";")[0].strip()
        if content_type not in VALID_CONTENT:
            raise ValueError(f"Invalid content type: {content_type}")
        return resp.content, content_type

    def _extract_colors(
        self, image: Image.Image, count: int = COLOR_LIMIT
    ) -> List[Color]:
        prepared_image = self._prepare_image_for_analysis(image)
        raw_colors = self._extract_raw_colors(prepared_image, count)
        return self._process_color_data(raw_colors)

    def _remove_border(self, image: Image.Image) -> Image.Image:
        bg = Image.new(image.mode, image.size, image.getpixel((0, 0)))
        diff = ImageChops.difference(image, bg)
        diff = ImageChops.add(diff, diff, 1.0, -100)
        bbox = diff.getbbox()
        return image.crop(bbox) if bbox else image

    def _prepare_image_for_analysis(self, image: Image.Image) -> Image.Image:
        resized, _, _ = self.resize_image(image, LOW_RES)
        return self._remove_border(resized)

    def _extract_raw_colors(
        self, image: Image.Image, count: int
    ) -> List[Tuple[Tuple[int, int, int], int]]:
        colors, _ = extcolors.extract_from_image(
            img=image, tolerance=COLOR_TOLERANCE, limit=count
        )
        return colors

    def _process_color_data(
        self, raw_colors: List[Tuple[Tuple[int, int, int], int]]
    ) -> List[Color]:
        total_pixels = sum(pixels for _, pixels in raw_colors)
        processed_colors = []

        for rgb, pixels in raw_colors:
            hex_color = webcolors.rgb_to_hex(rgb)
            css_name = self._rgb_to_css(rgb)
            html_name = self._rgb_to_html(rgb)
            percent = round(pixels / total_pixels, 8)

            color = Color(hex=hex_color, css=css_name, html=html_name, percent=percent)
            color = self._override_color_names(color)
            processed_colors.append(color)

        return processed_colors

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

    def _rgb_to_css(self, rgb: Tuple[int, int, int]) -> str:
        _, index = self._css_color_tree.query(rgb)
        return self._css_names[index]

    def _rgb_to_html(self, rgb: Tuple[int, int, int]) -> str:
        _, index = self._html_color_tree.query(rgb)
        return self._html_names[index]

    def _override_color_names(self, color: Color) -> Color:
        if color.html in {"navy", "purple"}:
            return color

        if color.css in self.CSS_OVERRIDES:
            color.html = self.CSS_OVERRIDES[color.css]

        if color.html in self.HTML_OVERRIDES:
            color.html = self.HTML_OVERRIDES[color.html]

        return color

    def _build_css_color_tree(self):
        rgb_values = []
        for name in self._css_names:
            hex_color = webcolors.name_to_hex(name, spec=webcolors.CSS3)
            rgb_values.append(webcolors.hex_to_rgb(hex_color))
        return KDTree(rgb_values)

    def _build_html_color_tree(self):
        rgb_values = []
        for name in self._html_names:
            hex_color = webcolors.name_to_hex(name, spec=webcolors.HTML4)
            rgb_values.append(webcolors.hex_to_rgb(hex_color))
        return KDTree(rgb_values)

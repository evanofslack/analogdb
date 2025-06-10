from io import BytesIO
from typing import Dict, List, Optional, Tuple

import extcolors
import numpy as np
import requests
import webcolors
from PIL import Image, ImageChops
from retry import retry
from scipy.spatial import KDTree

from .constants import COLOR_LIMIT, COLOR_TOLERANCE, LOW_RES
from .models import Color


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

    def __init__(self):
        self._css_color_tree = self._build_css_color_tree()
        self._html_color_tree = self._build_html_color_tree()
        self._css_names = list(webcolors.names(spec=webcolors.CSS3))
        self._html_names = list(webcolors.names(spec=webcolors.HTML4))

    @retry(delay=1, tries=5)
    def download_image(self, url: str) -> Image.Image:
        resp = requests.get(url)
        resp.raise_for_status()
        buf = BytesIO(resp.content)
        return Image.open(buf)

    def is_grayscale(self, image: Image.Image) -> bool:
        img = image.convert("RGB")
        img_array = np.array(img)
        # Check R, G, B channels all equal
        r, g, b = img_array[:, :, 0], img_array[:, :, 1], img_array[:, :, 2]
        return np.array_equal(r, g) and np.array_equal(g, b)

    def resize_image(
        self, image: Image.Image, size: Optional[Tuple[int, int]]
    ) -> Tuple[Image.Image, int, int]:
        if not size:
            return image, image.width, image.height
        img_resized = image.copy()
        img_resized.thumbnail(size, Image.Resampling.LANCZOS)
        return img_resized, img_resized.width, img_resized.height

    def image_to_bytes(self, image: Image.Image, content_type: str) -> BytesIO:
        image_bytes = BytesIO()
        format_name = content_type.removeprefix("image/")
        image.save(image_bytes, format_name)
        image_bytes.seek(0)
        return image_bytes

    def extract_colors(
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

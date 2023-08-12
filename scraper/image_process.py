from io import BytesIO
from typing import Dict, List, Optional, Tuple

import extcolors
import requests
from PIL import ImageChops
from PIL.Image import ANTIALIAS, Image, new, open
from scipy.spatial import KDTree
from webcolors import (CSS3_HEX_TO_NAMES, HTML4_HEX_TO_NAMES, hex_to_rgb,
                       rgb_to_hex)

from api import retry
from constants import COLOR_LIMIT, COLOR_TOLERANCE, LOW_RES
from models import Color

htmlOverrides: Dict[str, str] = {
    "silver": "gray",
    "fuschia": "purple",
    "blue": "teal",
    "aqua": "teal",
}

cssOverrides: Dict[str, str] = {
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


@retry(delay=1, times=5)
def request_image(url: str) -> Image:
    pic = requests.get(url, stream=True)
    image = open(pic.raw)
    return image


def resize_image(
    image: Image, size: Optional[Tuple[int, int]]
) -> Tuple[Image, int, int]:
    if not size:  # raw image, don't resize
        return image, image.width, image.height
    img_resized = image.copy()
    img_resized.thumbnail(size, ANTIALIAS)
    w = img_resized.width
    h = img_resized.height
    return img_resized, w, h


def is_grayscale(image: Image) -> bool:
    img = image.convert("RGB")
    w, h = img.size
    for i in range(w):
        for j in range(h):
            r, g, b = img.getpixel((i, j))
            if r != g != b:
                return False
    return True


def image_to_bytes(image: Image, content_type: str) -> BytesIO:
    image_bytes = BytesIO()
    image.save(image_bytes, content_type.removeprefix("image/"))
    image_bytes.seek(0)
    return image_bytes


def remove_border(image: Image) -> Image:
    bg = new(image.mode, image.size, image.getpixel((0, 0)))
    diff = ImageChops.difference(image, bg)
    diff = ImageChops.add(diff, diff, 1.0, -100)
    bbox = diff.getbbox()
    if bbox:
        return image.crop(bbox)
    else:
        return image


def rgb_to_css(rgb: Tuple[int, int, int]) -> str:
    # use KDTree to find closest CSS name for RGB color

    names = []
    rgb_values = []

    for hex, name in CSS3_HEX_TO_NAMES.items():
        names.append(name)
        rgb_values.append(hex_to_rgb(hex))

    kdt_db = KDTree(rgb_values)
    _, index = kdt_db.query(rgb)
    match = names[index]
    return match


def rgb_to_html(rgb: Tuple[int, int, int]) -> str:
    # use KDTree to find closest HTML name for RGB color

    names = []
    rgb_values = []

    for hex, name in HTML4_HEX_TO_NAMES.items():
        names.append(name)
        rgb_values.append(hex_to_rgb(hex))

    kdt_db = KDTree(rgb_values)
    _, index = kdt_db.query(rgb)
    match = names[index]
    return match


def override_color_names(color: Color) -> Color:

    # don't override these colors
    if color.html in {"navy", "purple"}:
        return color

    css = color.css
    if css in cssOverrides.keys():
        new = cssOverrides.get(css)
        if new is not None:
            color.html = new

    html = color.html
    if html in htmlOverrides.keys():
        new = htmlOverrides.get(html)
        if new is not None:
            color.html = new

    return color


def extract_colors(image: Image, count: int = COLOR_LIMIT) -> List[Color]:
    """

    extracts primary colors and their percentages from image

    """

    # resize the image for faster processing
    resized, _, _ = resize_image(image=image, size=LOW_RES)
    # remove border if found
    trim = remove_border(resized)

    colors, total_pixels = extcolors.extract_from_image(
        img=trim, tolerance=COLOR_TOLERANCE, limit=count
    )

    extracted: List[Color] = []
    for rgb, pixels in colors:

        # convert color to hex
        hex = rgb_to_hex(rgb)

        # get closest matching css color
        css = rgb_to_css(rgb)

        # get closest matching html color
        html = rgb_to_html(rgb)

        # get percent of image with this color
        percent = round(pixels / total_pixels, 8)

        # create color
        color = Color(hex=hex, css=css, html=html, percent=percent)

        # override color names
        color = override_color_names(color)

        # append it
        extracted.append(color)

    return extracted


def test_extract_colors():

    url = "https://d3i73ktnzbi69i.cloudfront.net/9c995e5b-9307-4f51-a58b-170e41e5fef3.jpeg"
    im = request_image(url)

    extracted = extract_colors(im)
    for e in extracted:
        print(e)


if __name__ == "__main__":
    test_extract_colors()

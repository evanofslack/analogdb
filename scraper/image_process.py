from io import BytesIO
from typing import List, Optional, Tuple

import extcolors
import requests
from loguru import logger
from PIL import ImageChops
from PIL.Image import ANTIALIAS, Image, new, open
from scipy.spatial import KDTree
from webcolors import (CSS3_HEX_TO_NAMES, HTML4_HEX_TO_NAMES, hex_to_rgb,
                       rgb_to_hex)

from constants import COLOR_LIMIT, COLOR_TOLERANCE, LOW_RES
from models import Color


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

        # append it
        extracted.append(Color(hex=hex, css=css, html=html, percent=percent))

    # we need to send 5 colors to analogdb
    # if we dont have 5 colors, append fillers
    num_filler = COLOR_LIMIT - len(extracted)
    if num_filler > 0:
        filler = Color(hex="null", css="null", html="null", percent=0.0)
        for _ in range(num_filler):
            extracted.append(filler)

    return extracted


def test_extract_colors():
    import PIL

    url = "https://d3i73ktnzbi69i.cloudfront.net/98fe51da-4b04-47db-b529-ce94f2c31219.jpeg"
    im = PIL.Image.open(requests.get(url, stream=True).raw)

    extracted = extract_colors(im)
    for e in extracted:
        print(e)


if __name__ == "__main__":
    test_extract_colors()

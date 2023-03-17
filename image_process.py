from io import BytesIO
from typing import List, Optional, Tuple

import extcolors
from loguru import logger
from PIL.Image import ANTIALIAS, Image, open

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


def rgb2hex(r, g, b):
    def pre(x):
        # clamp, round and convert to int
        return max(0, min(int(round(x)), 255))

    return f"#{pre(r):02x}{pre(g):02x}{pre(b):02x}"


def extract_colors(image: Image, count: int = COLOR_LIMIT) -> List[Color]:
    """

    extracts primary colors and their percentages from image

    """

    # resize the image for faster processing
    resized, _, _ = resize_image(image=image, size=LOW_RES)

    colors, total_pixels = extcolors.extract_from_image(
        img=resized, tolerance=COLOR_TOLERANCE, limit=count
    )

    extracted: List[Color] = []
    for rgb, pixels in colors:

        # convert color to hex
        hex = rgb2hex(*rgb)

        # get percent of image with this color
        percent = (pixels / total_pixels) * 100

        # append it
        extracted.append(Color(hex=hex, percent=percent))

    return extracted


def test_extract_colors():
    import PIL
    import requests

    url = "https://d3i73ktnzbi69i.cloudfront.net/0beebf59-4b2d-461e-b261-afcd19c51064.jpeg"
    im = PIL.Image.open(requests.get(url, stream=True).raw)

    extracted = extract_colors(im)
    for e in extracted:
        print(e)


if __name__ == "__main__":
    test_extract_colors()

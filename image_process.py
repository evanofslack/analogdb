from io import BytesIO
from typing import Optional, Tuple

import requests
from loguru import logger
from PIL.Image import ANTIALIAS, Image, open


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

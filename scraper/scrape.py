import base64
from io import BytesIO
from typing import List, Tuple

import praw
import requests
from PIL import Image, UnidentifiedImageError


def to_base64(img: Image) -> str:
    with BytesIO() as buffered:
        img.save(buffered, format="JPEG")
        img_str = base64.b64encode(buffered.getvalue())
        return img_str


def to_image(url: str) -> Image:
    pic = requests.get(url, stream=True)
    img = Image.open(pic.raw)
    # img.show()
    return img


def get_pics() -> List[Tuple[str, str]]:
    reddit = praw.Reddit("bot_1")
    pic_data: List[Tuple[str, str]] = []
    submissions = [s for s in reddit.subreddit("analog").hot(limit=5) if not s.is_self]

    for s in submissions:
        try:
            img = to_image(s.url)
            base64 = to_base64(img)
            # pic_data.append((s.url, base64))
            pic_data.append((s.url, "test"))
        except UnidentifiedImageError:
            print("Could not process image")
            pass
    return pic_data

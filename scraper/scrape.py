import base64
from io import BytesIO
from typing import List, Tuple

import praw
import requests
from PIL import Image


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


def get_pics() -> Tuple[List[str], List[str]]:
    reddit = praw.Reddit("bot_1")
    urls = []
    blob = []

    for submission in reddit.subreddit("analog").hot(limit=5):
        if not submission.is_self:
            urls.append(submission.url)
            img = to_image(submission.url)
            base64 = to_base64(img)
            blob.append(base64)
    return urls, blob


if __name__ == "__main__":
    urls, blob = get_pics()
    for url in urls:
        print(url)

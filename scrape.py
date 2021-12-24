import base64
import os
from dataclasses import dataclass
from io import BytesIO
from typing import List, Tuple

import praw
import requests
from PIL import Image


@dataclass
class AnalogData:
    title: str
    url: str
    permalink: str
    score: int
    nsfw: bool
    time: str


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


def handle_gallery(s: praw.reddit.Submission) -> str:
    """
    Return just the first image of the gallery

    """
    print("\nHandling Gallery")
    for item in sorted(s.gallery_data["items"], key=lambda x: x["id"]):
        media_id = item["media_id"]
        meta = s.media_metadata[media_id]
        if meta["e"] == "Image":
            source = meta["s"]
            return source["u"]


def get_url(s: praw.reddit.Submission) -> str:

    if hasattr(s, "is_gallery"):
        if s.is_gallery:
            return handle_gallery(s)
    else:
        return s.url


def append_link(path: str) -> str:
    return "https://www.reddit.com" + path


def get_pics() -> List[AnalogData]:
    # reddit = praw.Reddit("bot_1")
    reddit = praw.Reddit(
        client_id=os.getenv["client_id"],
        client_secret=os.getenv["client_secret"],
        user_agent=os.getenv["user_agent"],
    )

    pic_data: List[AnalogData] = []
    submissions: List[praw.reddit.Submission] = [
        s for s in reddit.subreddit("analog").hot(limit=5) if not s.is_self
    ]

    for s in submissions:
        new_pic = AnalogData(
            url=get_url(s),
            title=s.title,
            permalink="https://www.reddit.com" + s.permalink,
            score=s.score,
            nsfw=s.over_18,
            time=s.created_utc,
        )
        print("\n", new_pic)
        pic_data.append(new_pic)

    return pic_data


if __name__ == "__main__":
    get_pics()

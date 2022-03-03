import base64
import os
from dataclasses import dataclass
from io import BytesIO
from typing import List

import praw
import requests
from PIL import Image


@dataclass
class AnalogData:
    url: str
    title: str
    author: str
    permalink: str
    score: int
    nsfw: bool
    greyscale: bool
    time: float
    width: int
    height: int
    sprocket: bool

    low_url: str
    low_width: int
    low_height: int
    med_url: str
    med_width: int
    med_height: int
    high_url: str
    high_width: int
    high_height: int


def to_base64(img: Image) -> str:
    with BytesIO() as buffered:
        img.save(buffered, format="JPEG")
        img_str = base64.b64encode(buffered.getvalue())
        return img_str


def to_image(url: str) -> Image:

    LOW_RES = [320, 320]
    MEDIUM_RES = [768, 768]
    HIGH_RES = [1200, 1200]

    pic = requests.get(url, stream=True)
    img = Image.open(pic.raw)

    low = resize_image(img, LOW_RES)
    med = resize_image(img, MEDIUM_RES)
    high = resize_image(img, HIGH_RES)
    raw = img

    return low, med, high, raw


def resize_image(img: Image, size: List[int]):
    img_resized = img.copy()
    img_resized.thumbnail(size, Image.ANTIALIAS)
    return img_resized


def is_greyscale(img: Image, subreddit: str):
    if subreddit == "analog_bw":
        return True
    img = img.convert("RGB")
    w, h = img.size
    for i in range(w):
        for j in range(h):
            r, g, b = img.getpixel((i, j))
            if r != g != b:
                return False
    return True


def handle_gallery(s: praw.reddit.Submission) -> str:
    """
    Return the first image of a gallery

    """
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


def get_pics(num_pics: int, subreddit: str) -> List[AnalogData]:
    reddit = praw.Reddit(
        client_id=os.environ.get("client_id"),
        client_secret=os.environ.get("client_secret"),
        user_agent=os.environ.get("user_agent"),
    )
    print(f"Scraping pictures from {subreddit}")
    pic_data: List[AnalogData] = []
    submissions: List[praw.reddit.Submission] = [
        s for s in reddit.subreddit(subreddit).hot(limit=num_pics) if not s.is_self
    ]
    print(f"Gathered {len(submissions)} posts")

    for s in submissions:
        try:
            url = get_url(s)
            low, med, high, raw = to_image(url)
            # TODO s3 upload here

            new_pic = AnalogData(
                url=url,
                title=s.title,
                author="u/" + s.author.name,
                permalink="https://www.reddit.com" + s.permalink,
                score=s.score,
                nsfw=s.over_18,
                greyscale=is_greyscale(raw, subreddit),
                time=int(s.created_utc),
                width=raw.size[0],
                height=raw.size[1],
                sprocket=True if subreddit == "SprocketShots" else False,
                low_url=low,
                low_width=low.size[0],
                low_height=low.size[1],
                med_url=med,
                med_width=med.size[0],
                med_height=med.size[1],
                high_url=high,
                high_width=high.size[0],
                high_height=high.size[1],
            )
            print(new_pic.title)
            pic_data.append(new_pic)

        except Exception as e:
            print(e)

    return pic_data


if __name__ == "__main__":
    get_pics(3, "analog")

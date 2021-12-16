import base64
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
    nsfw: bool


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


def handle_image(s: praw.reddit.Submission) -> str:
    return s.url


def handle_gallery(s: praw.reddit.Submission) -> str:
    """
    Return just the first image of the gallery

    """
    for item in sorted(s.gallery_data["items"], key=lambda x: x["id"]):
        media_id = item["media_id"]
        meta = s.media_metadata[media_id]
        if meta["e"] == "Image":
            source = meta["s"]
            return source


def get_url(s: praw.reddit.Submission) -> str:
    if hasattr(s, "is_picture"):
        if s.is_picture:
            return handle_image(s)
    if hasattr(s, "is_gallery"):
        if s.is_gallery:
            return handle_gallery(s)


def get_pics() -> List[AnalogData]:
    reddit = praw.Reddit("bot_1")
    pic_data: List[AnalogData] = []
    submissions: List[praw.reddit.Submission] = [
        s for s in reddit.subreddit("analog").hot(limit=5) if not s.is_self
    ]

    for s in submissions:
        new_pic = AnalogData(
            title=s.title,
            url=get_url(s),
            permalink=s.permalink,
            nsfw=s.over_18,
        )
        print(new_pic)
        pic_data.append(new_pic)

    return pic_data


if __name__ == "__main__":
    get_pics()

    # reddit = praw.Reddit("bot_1")
    # url = "https://www.reddit.com/gallery/rhdv1w"
    # submission = reddit.submission(url=url)

    # print(submission.gallery_data["items"])
    # print("\n")
    # print(submission.media_metadata.keys())
    # meta = submission.media_metadata.pop()
    # if meta["e"] == "Image":
    #     print(meta["s"])

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
    submissions: List[praw.reddit.Submission] = [
        s for s in reddit.subreddit("analog").hot(limit=5) if not s.is_self
    ]

    for s in submissions:
        print("\nTitle:", s.title)
        # print("\nName:", s.name)
        # print("\nPermalink:", s.permalink)
        # print("\nNum Comments:", s.num_comments)
        # print("\nNSFW:", s.over_18)
        # print(vars(s))

        try:
            print("\nPost Hint:", s.post_hint)
        except:
            pass
        try:
            print("\nDomain:", s.domain)
        except:
            pass
        try:
            print("\nGallery Data:", s.gallery_data)
        except:
            pass
        # try:
        #     print("\nMedia Metadata:", s.media_metadata)
        # except:
        #     pass
        try:
            print("\nGallery:", s.is_gallery)
        except:
            pass
        try:
            print("\nPicture:", s.is_picture)
        except:
            pass

        pic_data.append((s.url, s.title))
    return pic_data


if __name__ == "__main__":
    get_pics()

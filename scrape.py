import uuid
from dataclasses import dataclass
from os import getenv
from typing import List, Optional, Tuple

import praw
import requests
from PIL import Image

from s3_upload import init_s3, s3_upload


@dataclass
class MyImage:
    image: Image.Image
    url: Optional[str]
    width: int
    height: float


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


def init_reddit() -> praw.Reddit:
    reddit = praw.Reddit(
        client_id=getenv("client_id"),
        client_secret=getenv("client_secret"),
        user_agent=getenv("user_agent"),
    )
    return reddit


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


def is_greyscale(img: Image.Image, subreddit: str):
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


def resize_image(img: Image.Image, size: List[int]):
    img_resized = img.copy()
    img_resized.thumbnail(size, Image.ANTIALIAS)
    w = img_resized.width
    h = img_resized.height
    return img_resized, w, h


def create_filename(url: str) -> Tuple[str, str]:
    viable_content = {
        "image/png": ".png",
        "image/jpeg": ".jpeg",
        "image/jpg": ".jpg",
        "image/gif": ".gif",
    }
    req = requests.get(url, stream=True)

    content_type = req.headers["content-type"]
    if content_type not in viable_content.keys():
        print(f"Cannot process {url} with type {content_type}")
        return
    filename = str(uuid.uuid4())
    filename += viable_content[content_type]
    return filename, content_type


def url_to_images(url: str, s3) -> List[MyImage]:
    """
    Download image from URL, create 3 new resolutions, and upload to S3

    """

    LOW_RES = [320, 320]
    MEDIUM_RES = [768, 768]
    HIGH_RES = [1200, 1200]
    RAW = "RAW"
    resolutions = [LOW_RES, MEDIUM_RES, HIGH_RES, RAW]
    BUCKET = "analog-photos-test"

    pic = requests.get(url, stream=True)
    img = Image.open(pic.raw)

    images: List[MyImage] = []
    for res in resolutions:

        if res == "RAW":
            f, c = create_filename(url)
            new_url = s3_upload(s3, bucket=BUCKET, image=i, filename=f, content_type=c)
            image = MyImage(image=img, url=new_url, width=img.width, height=img.height)
        else:
            i, w, h = resize_image(img, res)
            f, c = create_filename(url)
            new_url = s3_upload(s3, bucket=BUCKET, image=i, filename=f, content_type=c)
            image = MyImage(image=i, url=new_url, width=w, height=h)

        images.append(image)

    return images


def get_pics(
    reddit: praw.Reddit,
    s3,
    num_pics: int,
    subreddit: str,
    latest: List[str],
) -> List[AnalogData]:
    pic_data: List[AnalogData] = []
    submissions: List[praw.reddit.Submission] = [
        s for s in reddit.subreddit(subreddit).hot(limit=num_pics) if not s.is_self
    ]
    print(f"Gathered {len(submissions)} posts from {subreddit}")

    for s in submissions:
        if s.title in latest:
            # Don't upload post if it already exists in database
            print(f"duplicate post ({s.title})")
            continue

        try:
            url = get_url(s)
            images: List[MyImage] = url_to_images(url, s3)

            new_pic = AnalogData(
                url=images[3].url,
                title=s.title,
                author="u/" + s.author.name,
                permalink="https://www.reddit.com" + s.permalink,
                score=s.score,
                nsfw=s.over_18,
                greyscale=is_greyscale(images[3].image, subreddit),
                time=int(s.created_utc),
                width=images[3].width,
                height=images[3].height,
                sprocket=True if subreddit == "SprocketShots" else False,
                low_url=images[0].url,
                low_width=images[0].width,
                low_height=images[0].height,
                med_url=images[1].url,
                med_width=images[1].width,
                med_height=images[1].height,
                high_url=images[2].url,
                high_width=images[2].width,
                high_height=images[2].height,
            )
            print(new_pic.title)
            pic_data.append(new_pic)

        except Exception as e:
            print(f'Could not handle "{s.title}" at {url} with error: {e} ')

    return pic_data


if __name__ == "__main__":
    s3 = init_s3()
    reddit = init_reddit()
    get_pics(reddit, s3, 4, "analog")

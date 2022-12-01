from typing import List

import requests
from requests.auth import HTTPBasicAuth

from configuration import Config
from constants import ANALOGDB_URL
from scrape import AnalogPost


def get_latest() -> List[str]:
    url = f"{ANALOGDB_URL}/posts/latest?page_size=30"
    r = requests.get(url=url)
    data = r.json()
    posts = data["posts"]
    latest_titles = [post["title"] for post in posts]

    return latest_titles


def upload_post(config: Config, post: AnalogPost):
    data = post_to_json(post)
    url = f"{ANALOGDB_URL}/post"
    r = requests.put(
        url=url,
        data=data,
        auth=HTTPBasicAuth(config.auth.username, config.auth.password),
    )


def post_to_json(post: AnalogPost):
    images = post_to_images(post)
    body = {
        "images": images,
        "title": post.title,
        "author": post.author,
        "permalink": post.permalink,
        "upvotes": post.score,
        "nsfw": post.nsfw,
        "grayscale": post.greyscale,
        "unix_time": post.time,
        "sprocket": post.sprocket,
    }
    return body


def post_to_images(post: AnalogPost) -> List[dict]:
    # expected order is low, med, high, raw
    low = {
        "resolution": "low",
        "url": post.low_url,
        "width": post.low_width,
        "height": post.low_height,
    }
    med = {
        "resolution": "medium",
        "url": post.med_url,
        "width": post.med_width,
        "height": post.med_height,
    }
    high = {
        "resolution": "high",
        "url": post.high_url,
        "width": post.high_width,
        "height": post.high_height,
    }
    raw = {
        "resolution": "raw",
        "url": post.url,
        "width": post.width,
        "height": post.height,
    }
    return [low, med, high, raw]

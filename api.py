import json
from typing import List

import requests
from loguru import logger
from requests.auth import HTTPBasicAuth

from constants import ANALOGDB_URL
from models import AnalogPost


@logger.catch
def get_latest_links() -> List[str]:
    url = f"{ANALOGDB_URL}/posts/latest?page_size=100"
    r = requests.get(url=url)
    data = r.json()
    posts = data["posts"]
    latest_links = [post["permalink"] for post in posts]
    return latest_links


@logger.catch
def upload_to_analogdb(post: AnalogPost, username: str, password: str):
    dict_post = post_to_json(post)
    json_post = json.dumps(dict_post)
    url = f"{ANALOGDB_URL}/post"
    resp = requests.put(
        url=url,
        data=json_post,
        auth=HTTPBasicAuth(username=username, password=password),
    )

    code = resp.status_code
    msg = json.dumps(resp.content, indent=1)
    if code == 201:
        logger.info(
            f"created post with title: {post.title} with status code: {code} and msg: {msg}"
        )
    else:
        logger.error(
            f"failed to create post with title: {post.title} with status code: {code} and msg: {msg}"
        )


def delete_from_analogdb(id: int, username: str, password: str):
    url = f"{ANALOGDB_URL}/post/{id}"
    resp = requests.delete(
        url=url,
        auth=HTTPBasicAuth(username=username, password=password),
    )
    if resp.status_code == 200:
        logger.info("deleted post")
    else:
        logger.error("failed to delete post")


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

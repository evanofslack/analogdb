import json
from typing import List, Optional

import requests
from loguru import logger
from requests.auth import HTTPBasicAuth

from constants import ANALOGDB_URL
from models import AnalogDisplayPost, AnalogPost, PatchPost


def get_latest_posts(count: int) -> List[AnalogDisplayPost]:
    url = f"{ANALOGDB_URL}/posts?sort=latest&page_size={count}"
    try:
        r = requests.get(url=url)
    except Exception as e:
        raise Exception(f"Error making get request to analogdb: {e}")
    try:
        data = r.json()
    except Exception as e:
        raise Exception(f"Error unmarshalling json from analogdb: {e}")

    json_posts = data["posts"]

    analog_posts: List[AnalogDisplayPost] = []
    for json_post in json_posts:
        post = json_to_post(json_post)
        analog_posts.append(post)

    return analog_posts


def get_latest_links() -> List[str]:
    posts = get_latest_posts(count=100)
    links = [post.permalink for post in posts]
    return links


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
    msg = json.loads(resp.text)
    if code == 201:
        logger.info(
            f"created post with title: {post.title} with status code: {code} and msg: {msg}"
        )
    else:
        logger.error(
            f"failed to create post with title: {post.title} with status code: {code} and msg: {msg}"
        )


def patch_to_analogdb(patch: PatchPost, id: int, username: str, password: str):
    dict_patch = patch_to_json(patch)
    json_patch = json.dumps(dict_patch)
    url = f"{ANALOGDB_URL}/post/{id}"
    resp = requests.patch(
        url=url,
        data=json_patch,
        auth=HTTPBasicAuth(username=username, password=password),
    )
    if resp.status_code == 200:
        logger.info("patched post")
    else:
        logger.error("failed to patch post")


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


def json_to_post(data: dict) -> AnalogDisplayPost:

    post = AnalogDisplayPost(
        id=data["id"],
        title=data["title"],
        author=data["author"],
        permalink=data["permalink"],
        score=data["score"],
        nsfw=data["nsfw"],
        grayscale=data["grayscale"],
        timestamp=data["timestamp"],
        sprocket=data["sprocket"],
        low_url=data["images"][0]["url"],
        low_width=data["images"][0]["width"],
        low_height=data["images"][0]["height"],
        med_url=data["images"][1]["url"],
        med_width=data["images"][1]["width"],
        med_height=data["images"][1]["height"],
        high_url=data["images"][2]["url"],
        high_width=data["images"][2]["width"],
        high_height=data["images"][2]["height"],
        raw_url=data["images"][3]["url"],
        raw_width=data["images"][3]["width"],
        raw_height=data["images"][3]["height"],
    )

    return post


def post_to_json(post: AnalogPost):
    images = post_to_json_images(post)
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


def post_to_json_images(post: AnalogPost) -> List[dict]:
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


def patch_to_json(patch: PatchPost):
    body = {}
    if patch.score is not None:
        body["upvotes"] = patch.score
    if patch.nsfw is not None:
        body["nsfw"] = patch.nsfw
    if patch.greyscale is not None:
        body["grayscale"] = patch.greyscale
    if patch.sprocket is not None:
        body["sprocket"] = patch.sprocket
    return body


def new_patch(
    score: Optional[int] = None,
    nsfw: Optional[bool] = None,
    greyscale: Optional[bool] = None,
    sprocket: Optional[bool] = None,
) -> PatchPost:
    patch = PatchPost(score=score, nsfw=nsfw, greyscale=greyscale, sprocket=sprocket)
    return patch

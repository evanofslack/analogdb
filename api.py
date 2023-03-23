import json
from typing import List, Optional

import requests
from loguru import logger
from requests.auth import HTTPBasicAuth

from constants import ANALOGDB_URL
from models import AnalogDisplayPost, AnalogPost, Color, PatchPost


def get_latest_posts(count: int) -> List[AnalogDisplayPost]:

    # max page size is 200
    url = f"{ANALOGDB_URL}/posts?sort=latest&page_size={count}"
    analog_posts: List[AnalogDisplayPost] = []

    # loop until all pages have been queried
    while len(analog_posts) < count:
        try:
            r = requests.get(url=url)
        except Exception as e:
            raise Exception(f"Error making get request to analogdb: {e}")
        try:
            data = r.json()
        except Exception as e:
            raise Exception(f"Error unmarshalling json from analogdb: {e}")

        json_posts = data["posts"]

        for json_post in json_posts:
            post = json_to_post(json_post)
            analog_posts.append(post)

        meta = data["meta"]
        url = meta["next_page_url"]
        if url == "":
            break

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
    if resp.status_code != 200:
        raise Exception(f"failed to patch post with response: {resp.content}")


def delete_from_analogdb(id: int, username: str, password: str):
    url = f"{ANALOGDB_URL}/post/{id}"
    resp = requests.delete(
        url=url,
        auth=HTTPBasicAuth(username=username, password=password),
    )
    if resp.status_code != 200:
        raise Exception(f"failed to delete post with response: {resp.json()}")


def json_to_post(data: dict) -> AnalogDisplayPost:

    try:

        images = data["images"]
        colors = data["colors"]

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
            low_url=images[0]["url"],
            low_width=images[0]["width"],
            low_height=images[0]["height"],
            med_url=images[1]["url"],
            med_width=images[1]["width"],
            med_height=images[1]["height"],
            high_url=images[2]["url"],
            high_width=images[2]["width"],
            high_height=images[2]["height"],
            raw_url=images[3]["url"],
            raw_width=images[3]["width"],
            raw_height=images[3]["height"],
            c1_hex=colors[0]["hex"],
            c1_css=colors[0]["css"],
            c1_percent=colors[0]["percent"],
            c2_hex=colors[0]["hex"],
            c2_css=colors[0]["css"],
            c2_percent=colors[0]["percent"],
            c3_hex=colors[0]["hex"],
            c3_css=colors[0]["css"],
            c3_percent=colors[0]["percent"],
            c4_hex=colors[0]["hex"],
            c4_css=colors[0]["css"],
            c4_percent=colors[0]["percent"],
            c5_hex=colors[0]["hex"],
            c5_css=colors[0]["css"],
            c5_percent=colors[0]["percent"],
        )
    except Exception as e:
        raise Exception(f"Error unmarshalling json posts from analogdb: {e}")

    return post


def post_to_json(post: AnalogPost):
    images = post_to_json_images(post)
    colors = post_to_json_colors(post)
    body = {
        "title": post.title,
        "author": post.author,
        "permalink": post.permalink,
        "upvotes": post.score,
        "nsfw": post.nsfw,
        "grayscale": post.greyscale,
        "unix_time": post.time,
        "sprocket": post.sprocket,
        "images": images,
        "colors": colors,
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


def post_to_json_colors(post: AnalogPost) -> List[dict]:
    # expected 5 colors
    c1 = {
        "hex": post.c1_hex,
        "css": post.c1_css,
        "percent": post.c1_percent,
    }
    c2 = {
        "hex": post.c2_hex,
        "css": post.c2_css,
        "percent": post.c2_percent,
    }
    c3 = {
        "hex": post.c3_hex,
        "css": post.c3_css,
        "percent": post.c3_percent,
    }
    c4 = {
        "hex": post.c4_hex,
        "css": post.c4_css,
        "percent": post.c4_percent,
    }
    c5 = {
        "hex": post.c5_hex,
        "css": post.c5_css,
        "percent": post.c5_percent,
    }
    return [c1, c2, c3, c4, c5]


def colors_to_json(colors: List[Color]) -> List[dict]:
    # expected 5 colors from highest to lowest percent
    json_colors = []
    for c in colors:
        temp = {"hex": c.hex, "css": c.css, "percent": c.percent}
        json_colors.append(temp)
    return json_colors


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
    if patch.colors is not None:
        body["colors"] = colors_to_json(colors=patch.colors)
    return body


def new_patch(
    score: Optional[int] = None,
    nsfw: Optional[bool] = None,
    greyscale: Optional[bool] = None,
    sprocket: Optional[bool] = None,
    colors: Optional[List[Color]] = None,
) -> PatchPost:
    patch = PatchPost(
        score=score, nsfw=nsfw, greyscale=greyscale, sprocket=sprocket, colors=colors
    )
    return patch

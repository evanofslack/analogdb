import functools
import json
import time
from typing import List, Optional

import requests
from loguru import logger
from requests.auth import HTTPBasicAuth

from configuration import init_config
from models import (AnalogDisplayPost, AnalogKeyword, AnalogPost, Color,
                    PatchPost)

config = init_config()
base_url = config.app.api_base_url

# decorator to retry operations
def retry(delay=1, times=5):
    def outer_wrapper(function):
        @functools.wraps(function)
        def inner_wrapper(*args, **kwargs):
            final_excep = None
            for counter in range(times):
                if counter > 0:
                    time.sleep(delay)
                final_excep = None
                try:
                    value = function(*args, **kwargs)
                    return value
                except Exception as e:
                    final_excep = e
                    logger.warning(
                        f"Error during function call, count={counter}: error:{e}"
                    )

            if final_excep is not None:
                raise final_excep

        return inner_wrapper

    return outer_wrapper


def get_latest_posts(count: int) -> List[AnalogDisplayPost]:

    # max page size is 200
    url = f"{base_url}/posts?sort=latest&page_size={count}"
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
    url = f"{base_url}/post"
    resp = requests.put(
        url=url,
        data=json_post,
        auth=HTTPBasicAuth(username=username, password=password),
    )
    code = resp.status_code
    msg = json.loads(resp.text)
    if code == 201:
        logger.info(f"created post (title: {post.title} | status: {code} | msg: {msg})")
    else:
        logger.error(
            f"failed to create post (title: {post.title} | status: {code} | msg: {msg})"
        )


@retry(delay=1, times=5)
def patch_to_analogdb(patch: PatchPost, id: int, username: str, password: str):
    dict_patch = patch_to_json(patch)
    json_patch = json.dumps(dict_patch)
    url = f"{base_url}/post/{id}"
    resp = requests.patch(
        url=url,
        data=json_patch,
        auth=HTTPBasicAuth(username=username, password=password),
    )
    if resp.status_code != 200:
        raise Exception(f"failed to patch post with response: {resp.content}")


def delete_from_analogdb(id: int, username: str, password: str):
    url = f"{base_url}/post/{id}"
    resp = requests.delete(
        url=url,
        auth=HTTPBasicAuth(username=username, password=password),
    )
    if resp.status_code != 200:
        raise Exception(f"failed to delete post with response: {resp.json()}")


def get_all_post_ids() -> List[int]:
    url = f"{base_url}/ids"
    resp = requests.get(
        url=url,
    )
    if resp.status_code != 200:
        raise Exception(f"failed to delete post with response: {resp.json()}")

    json_ids = resp.json()["ids"]

    return [int(id) for id in json_ids]


def get_keyword_updated_post_ids(username: str, password: str) -> List[int]:

    path = "scrape/keywords/updated"

    url = f"{base_url}/{path}"
    r = requests.get(
        url=url,
        auth=HTTPBasicAuth(username=username, password=password),
    )
    if r.status_code != 200:
        raise Exception(f"failed to fetch {path} with response: {r.json()}")
    try:
        data = r.json()
    except Exception as e:
        raise Exception(f"Error unmarshalling json from analogdb: {e}")

    ids = data["ids"]

    return ids


# type encodePostsRequest struct {
# 	Ids       []int `json:"ids"`
# 	BatchSize int   `json:"batch_size"`
# }
#


def encode_images(ids: List[int], batch_size: int, username: str, password: str):
    data = {"ids": ids, "batch_size": batch_size}
    body = json.dumps(data)
    url = f"{base_url}/encode"
    logger.info(f"encoding post ids {ids}")
    resp = requests.put(
        url=url,
        data=body,
        auth=HTTPBasicAuth(username=username, password=password),
    )
    if resp.status_code != 200:
        raise Exception(f"failed encode posts {ids} with response: {resp.content}")


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
        )
    except Exception as e:
        raise Exception(f"Error unmarshalling json posts from analogdb: {e}")

    return post


def post_to_json(post: AnalogPost):
    images = post_to_json_images(post)
    colors = colors_to_json(post.colors)
    keywords = keywords_to_json(post.keywords)
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
        "keywords": keywords,
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


def keywords_to_json(keywords: List[AnalogKeyword]) -> List[dict]:

    json_keywords: List[dict] = []
    for kw in keywords:
        json_keywords.append({"word": kw.word, "weight": kw.weight})

    return json_keywords


def colors_to_json(colors: List[Color]) -> List[dict]:
    # expected 5 colors from highest to lowest percent
    json_colors = []
    for c in colors:
        temp = {"hex": c.hex, "css": c.css, "html": c.html, "percent": c.percent}
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
    if patch.keywords is not None:
        body["keywords"] = keywords_to_json(keywords=patch.keywords)
    return body


def new_patch(
    score: Optional[int] = None,
    nsfw: Optional[bool] = None,
    greyscale: Optional[bool] = None,
    sprocket: Optional[bool] = None,
    colors: Optional[List[Color]] = None,
    keywords: Optional[List[AnalogKeyword]] = None,
) -> PatchPost:
    patch = PatchPost(
        score=score,
        nsfw=nsfw,
        greyscale=greyscale,
        sprocket=sprocket,
        colors=colors,
        keywords=keywords,
    )
    return patch

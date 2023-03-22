from typing import Callable, List

import praw
import requests
from loguru import logger

from api import get_latest_posts, json_to_post, new_patch, patch_to_analogdb
from constants import ANALOGDB_URL
from models import AnalogDisplayPost


# takes a modifier function and applies it each post
# in an arbitrary amount of recent posts
def apply_to_recent_posts(
    modifier: Callable[[praw.Reddit, AnalogDisplayPost, str, str], None],
    count: int,
    reddit: praw.Reddit,
    username: str,
    password: str,
):
    # max page size is 200
    url = f"{ANALOGDB_URL}/posts?sort=latest&page_size={count}"
    seen = 0

    # loop until all pages have been queried
    while seen < count:
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
            modifier(reddit, post, username, password)

        meta = data["meta"]
        seen += int(meta["page_size"])
        next_page_url = meta["next_page_url"]

        url = f"{ANALOGDB_URL}{next_page_url}"
        if url == "":
            break


def update_post_score(
    reddit: praw.Reddit, post: AnalogDisplayPost, username: str, password: str
):
    url = post.permalink

    try:
        submission = reddit.submission(url=url)
        new_score = submission.score
    except Exception as e:
        raise Exception(
            f"Error fetching submission with url: {post.permalink}, with error: {e}"
        )

    # only update the score if the new score is higher than original
    if new_score <= post.score:
        logger.debug(f"post with ID: {post.id} does not have a higher score, skipping")
        return

    patch = new_patch(score=new_score)
    patch_to_analogdb(patch, id=post.id, username=username, password=password)
    logger.info(
        f"post with ID: {post.id} has score updated from {post.score} to {patch.score}"
    )


def update_latest_post_scores(
    reddit: praw.Reddit, count: int, username: str, password: str
):
    apply_to_recent_posts(
        modifier=update_post_score,
        count=count,
        reddit=reddit,
        username=username,
        password=password,
    )

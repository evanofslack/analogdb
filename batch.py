from typing import List, Optional, Set

import praw
import requests
from loguru import logger

from api import json_to_post, new_patch, patch_to_analogdb
from comment import post_keywords_from_disk, write_comments_to_json
from constants import ANALOGDB_URL
from image_process import extract_colors, request_image
from models import AnalogDisplayPost, Dependencies


def unlimited_posts(count: int) -> List[AnalogDisplayPost]:
    # max page size is 200
    # url = f"{ANALOGDB_URL}/posts?sort=latest&page_size={count}"
    url = f"{ANALOGDB_URL}/posts?sort=top&page_size={count}"

    posts: List[AnalogDisplayPost] = []

    # loop until all pages have been queried
    while len(posts) < count:
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
            posts.append(json_to_post(json_post))

        next_page_url = data["meta"]["next_page_url"]

        url = f"{ANALOGDB_URL}{next_page_url}"
        if url == "":
            break

    return posts


def _update_post_score(
    reddit: praw.Reddit, post: AnalogDisplayPost, username: str, password: str
):
    url = post.permalink

    try:
        submission = reddit.submission(url=url)
        new_score = submission.score
    except Exception as e:
        logger.error(
            f"Error fetching submission with url: {post.permalink}, with error: {e}"
        )
        return

    # only update the score if the new score is higher than original
    if new_score <= post.score:
        logger.debug(f"post with ID: {post.id} does not have a higher score, skipping")
        return

    patch = new_patch(score=new_score)
    patch_to_analogdb(patch, id=post.id, username=username, password=password)
    logger.info(
        f"post with ID: {post.id} has score updated from {post.score} to {patch.score}"
    )


def update_posts_scores(deps: Dependencies, count: int):
    posts = unlimited_posts(count=count)
    for post in posts:
        _update_post_score(
            reddit=deps.reddit_client,
            post=post,
            username=deps.auth.username,
            password=deps.auth.password,
        )


def _update_post_colors(
    reddit: praw.Reddit, post: AnalogDisplayPost, username: str, password: str
):
    url = post.low_url

    # extract primary colors
    try:
        image = request_image(url=url)
        colors = extract_colors(image)
    except Exception as e:
        logger.error(f"Error fetching iamge with url: {url}, with error: {e}")

    # update post in analogdb
    patch = new_patch(colors=colors)
    patch_to_analogdb(patch, id=post.id, username=username, password=password)
    logger.info(
        f"post with ID: {post.id} has colors updated to {[c.css for c in colors]}"
    )


def update_posts_colors(deps: Dependencies, count: int):
    posts = unlimited_posts(count=count)
    for post in posts:
        _update_post_colors(
            reddit=deps.reddit_client,
            post=post,
            username=deps.auth.username,
            password=deps.auth.password,
        )


def _download_post_comments(reddit: praw.Reddit, post: AnalogDisplayPost):
    try:
        write_comments_to_json(reddit=reddit, post=post)
    except Exception as e:
        logger.info(
            f"Error getting post comments for {post.permalink}, with error: {e}"
        )

    logger.info(f"saved post comments to comments/{post.id}.json")


def download_posts_comments(deps: Dependencies, count: int):
    posts = unlimited_posts(count=count)
    for post in posts:
        _download_post_comments(reddit=deps.reddit_client, post=post)


def _update_post_keywords(
    post: AnalogDisplayPost,
    limit: Optional[int] = None,
    blacklist: Optional[Set[str]] = None,
):
    try:
        keywords = post_keywords_from_disk(post=post, limit=limit, blacklist=blacklist)
        print(post.title)
        for kw in keywords:
            print(kw.word, kw.weight)

    except Exception as e:
        logger.info(f"Error reading json post comments for {post.id}, with error: {e}")


def update_posts_keywords(deps: Dependencies, count: int, limit: Optional[int] = None):
    posts = unlimited_posts(count=count)
    for post in posts:
        _update_post_keywords(post=post, limit=limit, blacklist=deps.blacklist)

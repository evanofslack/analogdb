import datetime
import time
from typing import List, Optional, Set

import praw
import requests
from loguru import logger

from api import (encode_images, get_all_post_ids, get_keyword_updated_post_ids,
                 json_to_post, new_patch, patch_to_analogdb)
from comment import (get_comments, post_keywords, read_comments_from_json,
                     write_comments_to_json, write_keywords_to_disk)
from configuration import init_config
from constants import (ALL_KEYWORDS_FILEPATH, KEYWORD_UPDATE_CUTOFF_DAYS,
                       READ_COMMENTS_FROM_DISK, WRITE_KEYWORDS_TO_DISK)
from image_process import extract_colors, request_image
from models import AnalogDisplayPost, Dependencies
from s3_upload import upload_comments_to_s3

config = init_config()
base_url = config.app.api_base_url


@logger.catch(message="caught error while getting posts from analogdb")
def unlimited_posts(count: int) -> List[AnalogDisplayPost]:
    # max page size is 200
    url = f"{base_url}/posts?sort=latest&page_size={count}"

    posts: List[AnalogDisplayPost] = []

    # loop until all pages have been queried
    while len(posts) < count:
        try:
            logger.debug(f"requesting {url}")
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

        url = f"{base_url}{next_page_url}"
        if next_page_url == "":
            logger.debug(f"done requesting, end of posts")

            break
        time.sleep(1)

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
    except Exception as e:
        logger.error(f"Error fetching image with url: {url}, with error: {e}")
        return

    try:
        colors = extract_colors(image)
    except Exception as e:
        logger.error(f"Error extracting colors from image: {url}, with error: {e}")
        return

    # update post in analogdb
    patch = new_patch(colors=colors)
    patch_to_analogdb(patch, id=post.id, username=username, password=password)
    logger.info(
        f"post with ID: {post.id} has colors updated to {[c.css for c in colors]}"
    )


def update_posts_colors(deps: Dependencies, count: int):
    posts = reversed(unlimited_posts(count=count))
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
    reddit: praw.Reddit,
    s3,
    username: str,
    password: str,
    limit: Optional[int] = None,
    blacklist: Optional[Set[str]] = None,
):

    if READ_COMMENTS_FROM_DISK:
        filepath = f"comments/{post.id}.json"
        comments = read_comments_from_json(filepath=filepath)

    else:
        try:
            comments = get_comments(reddit=reddit, url=post.permalink)
        except Exception as e:
            logger.info(f"Error getting comments for {post.id}, with error: {e}")
            return

    keywords = post_keywords(
        title=post.title,
        comments=comments,
        post_score=post.score,
        limit=limit,
        blacklist=blacklist,
    )
    logger.debug(f"keywords for post {post.title}")
    logger.debug([f"{kw.word}, {kw.weight}" for kw in keywords])

    if WRITE_KEYWORDS_TO_DISK:
        write_keywords_to_disk(keywords=keywords, filepath=ALL_KEYWORDS_FILEPATH)

    # update post in analogdb
    patch = new_patch(keywords=keywords)
    patch_to_analogdb(patch, id=post.id, username=username, password=password)
    logger.info(f"updated post {post.id} with {len(keywords)} total keywords")

    # upload the comments as json to s3
    upload_comments_to_s3(s3=s3, comments=comments, filename=f"{post.id}.json")


def update_posts_keywords(deps: Dependencies, count: int, limit: Optional[int] = None):

    logger.debug("start update post keywords")

    posts = unlimited_posts(count=count)

    # don't update a post's keywords more than once
    updated_ids = set(
        get_keyword_updated_post_ids(
            username=deps.auth.username, password=deps.auth.password
        )
    )

    # only update keywords for posts older than 2 days
    cutoff = (
        datetime.datetime.fromtimestamp(time.time())
        - datetime.timedelta(days=KEYWORD_UPDATE_CUTOFF_DAYS)
    ).timestamp()

    for post in posts:
        # post has already been updated, skip
        if post.id in updated_ids:
            logger.debug(f"post {post.id} already keyword updated")
            continue
        # post is too new, skip
        if post.timestamp > cutoff:
            logger.debug(f"post {post.id} too new to update keywords")
            continue

        # otherwise, update it
        _update_post_keywords(
            post=post,
            reddit=deps.reddit_client,
            s3=deps.s3_client,
            username=deps.auth.username,
            password=deps.auth.password,
            limit=limit,
            blacklist=deps.blacklist,
        )


def update_post_similars_by_ids(deps: Dependencies, ids: List[int], batch_size: int):
    encode_images(
        ids=ids,
        batch_size=batch_size,
        username=deps.auth.username,
        password=deps.auth.password,
    )


def update_post_similars(deps: Dependencies, count: int, batch_size: int):
    ids = get_all_post_ids()
    ids.sort(reverse=True)

    # do not go out of bounds
    if count > len(ids):
        count = len(ids)

    logger.info(
        f"updating post similars for {count} posts with batch size {batch_size}"
    )
    ids = ids[:count]
    encode_images(
        ids=ids,
        batch_size=batch_size,
        username=deps.auth.username,
        password=deps.auth.password,
    )


def chunk_list(data: List, chunksize: int):
    for i in range(0, len(data), chunksize):
        yield data[i : i + chunksize]


def batch():
    from configuration import dependencies_from_config, init_config
    from log import init_logger

    init_logger(with_file=False, with_slack=False)
    config = init_config()
    deps = dependencies_from_config(config=config)

    ids = get_all_post_ids()
    ids.sort(reverse=True)

    for ids_chunk in chunk_list(data=ids, chunksize=10):
        update_post_similars_by_ids(deps=deps, ids=ids_chunk, batch_size=1)
        time.sleep(1)


if __name__ == "__main__":
    batch()

import uuid
from typing import List, Tuple, Optional

import praw
import requests
from loguru import logger
from PIL.Image import Image

from constants import BW_SUB, REDDIT_URL, SPROCKET_SUB, VALID_CONTENT
from image_process import is_grayscale, request_image
from models import RedditPost


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


def create_filename(url: str) -> Optional[Tuple[str, str]]:
    viable_content = {
        "image/png": ".png",
        "image/jpeg": ".jpeg",
        "image/jpg": ".jpg",
        "image/gif": ".gif",
    }
    req = requests.get(url, stream=True)

    content_type = req.headers["content-type"]
    if content_type is None:
        return
    if content_type not in viable_content.keys():
        logger.warning(f"Cannot process {url} with type {content_type}")
        return
    filename = str(uuid.uuid4())
    filename += viable_content[content_type]
    return filename, content_type


def get_content_type(url) -> Optional[str]:
    try:
        req = requests.get(url, stream=True)
    except Exception as e:
        logger.error(f"get request for url failed; url={url} error={e}")
        return None

    content_type = req.headers["content-type"]

    try:
        content_type = req.headers["content-type"]
    except Exception as e:
        logger.error(f"get content type from request failed; url={url} error={e}")
        return None

    return content_type


def is_post_grayscale(image: Image, subreddit: str):
    if subreddit == BW_SUB:
        return True
    return is_grayscale(image)


def is_post_sprocket(subreddit: str):
    return subreddit == SPROCKET_SUB


def get_posts(
    reddit: praw.Reddit,
    num_posts: int,
    subreddit: str,
    latest_permalinks: List[str],
) -> List[RedditPost]:

    # get posts that are not self-posts
    submissions: List[praw.reddit.Submission] = [
        s for s in reddit.subreddit(subreddit).hot(limit=num_posts) if not s.is_self
    ]
    logger.debug(f"gathered {len(submissions)} posts from {subreddit}")

    posts: List[RedditPost] = []
    for s in submissions:

        # check if duplicate post
        permalink = f"{REDDIT_URL}{s.permalink}"
        if permalink in latest_permalinks:
            continue

        url = get_url(s)
        content_type = get_content_type(url)

        if content_type not in VALID_CONTENT:
            logger.warning(f"cannot process {url} with type {content_type}")
            continue

        image = request_image(url)

        post = RedditPost(
            image=image,
            width=image.width,
            height=image.height,
            content_type=content_type,
            title=s.title,
            author=f"u/{s.author.name}",
            permalink=permalink,
            score=s.score,
            nsfw=s.over_18,
            greyscale=is_post_grayscale(image, subreddit),
            time=int(s.created_utc),
            sprocket=is_post_sprocket(subreddit),
        )
        posts.append(post)
    return posts

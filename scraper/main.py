import gc
import time

import schedule
from loguru import logger

from api import get_latest_links, upload_to_analogdb
from batch import (update_posts_colors, update_posts_keywords,
                   update_posts_scores)
from comment import get_comments, post_keywords
from configuration import dependencies_from_config, init_config
from constants import (ANALOG_POSTS, ANALOG_SUB, BW_POSTS, BW_SUB,
                       KEYWORD_LIMIT, SPROCKET_POSTS, SPROCKET_SUB)
from log import init_logger
from models import Dependencies
from s3_upload import create_analog_post, upload_images_to_s3
from scrape import get_posts


@logger.catch(message="caught error while scraping posts")
def scrape_posts(
    deps: Dependencies,
    subreddit: str,
    num_posts: int,
):
    reddit_client = deps.reddit_client
    s3_client = deps.s3_client
    auth = deps.auth

    logger.debug(f"scraping r/{subreddit}")

    recent_posts = get_posts(
        reddit=reddit_client,
        num_posts=num_posts,
        subreddit=subreddit,
        latest_permalinks=get_latest_links(),
    )

    if not recent_posts:
        logger.debug("no new posts to upload")
        return
    else:
        logger.info(f"uploading {len(recent_posts)} new posts from r/{subreddit}")

    for post in recent_posts:
        # upload images to s3
        cf_images = upload_images_to_s3(post=post, s3=s3_client)
        # parse comments from post
        comments = get_comments(reddit=reddit_client, url=post.permalink)
        # get keywords from comments
        keywords = post_keywords(
            title=post.title,
            comments=comments,
            post_score=post.score,
            limit=KEYWORD_LIMIT,
            blacklist=deps.blacklist,
        )
        # create and upload the post
        analog_post = create_analog_post(images=cf_images, post=post, keywords=keywords)
        upload_to_analogdb(
            post=analog_post, username=auth.username, password=auth.password
        )


def scrape_analog(deps: Dependencies):
    sub = ANALOG_SUB
    scrape_posts(deps=deps, subreddit=sub, num_posts=ANALOG_POSTS)


def scrape_bw(deps: Dependencies):
    sub = BW_SUB
    scrape_posts(deps=deps, subreddit=sub, num_posts=BW_POSTS)


def scrape_sprocket(deps: Dependencies):
    sub = SPROCKET_SUB
    scrape_posts(deps=deps, subreddit=sub, num_posts=SPROCKET_POSTS)


@logger.catch(message="caught error while updating post scores")
def update_scores(deps: Dependencies):
    update_posts_scores(deps=deps, count=100)


@logger.catch(message="caught error while updating post keywords")
def update_keywords(deps: Dependencies):
    update_posts_keywords(deps=deps, count=100, limit=KEYWORD_LIMIT)


@logger.catch(message="caught error while updating post colors")
def update_colors(deps: Dependencies):
    update_posts_colors(deps=deps, count=100)


def run_schedule(deps: Dependencies):

    # scrape posts
    schedule.every().day.do(scrape_bw, deps=deps)
    schedule.every().day.do(scrape_sprocket, deps=deps)
    schedule.every(4).hours.do(scrape_analog, deps=deps)

    # schedule.every().day.do(update_scores, deps=deps)
    # schedule.every().day.do(update_keywords, deps=deps)

    schedule.run_all()

    while True:
        schedule.run_pending()
        gc.collect()  # cleanup
        time.sleep(4 * 3600)  # sleep for 4 hours


def main():

    init_logger()
    config = init_config()
    deps = dependencies_from_config(config=config)

    run_schedule(deps=deps)


if __name__ == "__main__":
    main()

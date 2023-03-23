import time

import schedule
from loguru import logger

from api import delete_from_analogdb, get_latest_links, upload_to_analogdb
from batch import update_latest_post_colors, update_latest_post_scores
from configuration import dependencies_from_config, init_config
from constants import (ANALOG_POSTS, ANALOG_SUB, AWS_BUCKET, BW_POSTS, BW_SUB,
                       SPROCKET_POSTS, SPROCKET_SUB)
from log import init_logger
from models import Dependencies
from s3_upload import create_analog_post, upload_to_s3
from scrape import get_posts


def scrape_posts(
    deps: Dependencies,
    subreddit: str,
    num_posts: int,
):
    reddit_client = deps.reddit_client
    s3_client = deps.s3_client
    auth = deps.auth

    logger.info(f"scraping r/{subreddit}")

    recent_posts = get_posts(
        reddit=reddit_client,
        num_posts=num_posts,
        subreddit=subreddit,
        latest_permalinks=get_latest_links(),
    )

    if not recent_posts:
        logger.info("no new posts to upload")
        return
    else:
        logger.info(f"uploading {len(recent_posts)} new posts")

    for post in recent_posts:
        cf_images = upload_to_s3(post=post, s3=s3_client, bucket=AWS_BUCKET)
        analog_post = create_analog_post(images=cf_images, post=post)
        upload_to_analogdb(
            post=analog_post, username=auth.username, password=auth.password
        )


def scrape_analog(deps: Dependencies):
    scrape_posts(deps=deps, subreddit=ANALOG_SUB, num_posts=ANALOG_POSTS)


def scrape_bw(deps: Dependencies):
    scrape_posts(deps=deps, subreddit=BW_SUB, num_posts=BW_POSTS)


def scrape_sprocket(deps: Dependencies):
    scrape_posts(deps=deps, subreddit=SPROCKET_SUB, num_posts=SPROCKET_POSTS)


def delete_post():
    config = init_config()
    deps = dependencies_from_config(config=config)
    auth = deps.auth
    delete_from_analogdb(id=99999, username=auth.username, password=auth.password)


def update_post_score(deps: Dependencies):
    update_latest_post_scores(
        reddit=deps.reddit_client,
        count=100,
        username=deps.auth.username,
        password=deps.auth.password,
    )


def update_post_colors(deps: Dependencies):
    update_latest_post_colors(
        reddit=deps.reddit_client,
        count=5100,
        username=deps.auth.username,
        password=deps.auth.password,
    )


def run_schedule(deps: Dependencies):

    # scrape posts
    schedule.every().day.do(scrape_bw, deps=deps)
    schedule.every().day.do(scrape_sprocket, deps=deps)
    schedule.every(4).hours.do(scrape_analog, deps=deps)

    # update latest 100 post scores each day
    schedule.every().day.do(update_post_score, deps=deps)

    schedule.run_all()

    while True:

        try:
            schedule.run_pending()
        except Exception as e:
            logger.error(f"issue running schedued job: {e}")
        time.sleep(4 * 3600)  # sleep for 4 hours


def main():

    init_logger()
    config = init_config()
    deps = dependencies_from_config(config=config)

    run_schedule(deps=deps)


if __name__ == "__main__":
    main()

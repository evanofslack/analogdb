import time

import schedule

from api import get_latest, upload_to_analogdb
from configuration import dependencies_from_config, init_config
from constants import (
    ANALOG_POSTS,
    ANALOG_SUB,
    AWS_BUCKET,
    BW_POSTS,
    BW_SUB,
    SPROCKET_POSTS,
    SPROCKET_SUB,
)
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

    saved_posts = get_latest()  # posts already stored in analogdb
    recent_posts = get_posts(
        reddit=reddit_client,
        num_posts=num_posts,
        subreddit=subreddit,
    )

    unsaved_posts = [post for post in recent_posts if post.title not in saved_posts]

    for post in unsaved_posts:
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


def test():
    config = init_config()
    deps = dependencies_from_config(config=config)

    reddit_client = deps.reddit_client
    s3_client = deps.s3_client
    auth = deps.auth

    saved_posts = get_latest()  # posts already stored in analogdb

    recent_posts = get_posts(
        reddit=reddit_client,
        num_posts=3,
        subreddit=SPROCKET_SUB,
    )

    print(f"recent posts: {[post.title for post in recent_posts]}\n")

    unsaved_posts = [post for post in recent_posts if post.title not in saved_posts]

    print(f"unsaved posts: {[post.title for post in unsaved_posts]}\n")

    for post in unsaved_posts:
        cf_images = upload_to_s3(post=post, s3=s3_client, bucket=AWS_BUCKET)
        analog_post = create_analog_post(images=cf_images, post=post)
        upload_to_analogdb(
            post=analog_post, username=auth.username, password=auth.password
        )


def main():

    config = init_config()
    deps = dependencies_from_config(config=config)

    schedule.every().day.do(scrape_bw, deps=deps)
    schedule.every().day.do(scrape_sprocket, deps=deps)
    schedule.every(4).hours.do(scrape_analog, deps=deps)

    while True:
        schedule.run_pending()
        time.sleep(60)


if __name__ == "__main__":
    # main()

    test()

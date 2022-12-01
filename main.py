import time

import boto3.session
import praw
import schedule

from api import get_latest, upload_post
from configuration import init_config
from constants import (
    ANALOG_POSTS,
    ANALOG_SUB,
    BW_POSTS,
    BW_SUB,
    SPROCKET_POSTS,
    SPROCKET_SUB,
)
from s3_upload import init_s3
from scrape import get_posts, init_reddit


def scrape_posts(
    s3: boto3.session.Session, reddit: praw.Reddit, subreddit: str, num_posts: int
):
    latest_posts = get_latest()
    for post in get_posts(
        reddit=reddit,
        s3=s3,
        num_posts=num_posts,
        subreddit=subreddit,
        latest=latest_posts,
    ):
        if post.title not in latest_posts:
            upload_post(post)


def scrape_analog(s3: boto3.session.Session, reddit: praw.Reddit):
    scrape_posts(s3=s3, reddit=reddit, subreddit=ANALOG_SUB, num_posts=ANALOG_POSTS)


def scrape_bw(s3: boto3.session.Session, reddit: praw.Reddit):
    scrape_posts(s3=s3, reddit=reddit, subreddit=BW_SUB, num_posts=BW_POSTS)


def scrape_sprocket(s3: boto3.session.Session, reddit: praw.Reddit):
    scrape_posts(s3=s3, reddit=reddit, subreddit=SPROCKET_SUB, num_posts=SPROCKET_POSTS)


def main():

    config = init_config()
    s3 = init_s3(config=config)
    reddit = init_reddit(config=config)

    schedule.every().day.do(scrape_bw, s3=s3, reddit=reddit)
    schedule.every().day.do(scrape_sprocket, s3=s3, reddit=reddit)
    schedule.every(4).hours.do(scrape_analog, s3=s3, reddit=reddit)

    while True:
        schedule.run_pending()
        time.sleep(60)


if __name__ == "__main__":
    main()

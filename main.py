import dataclasses
import datetime as dt
from typing import List

import boto3.session
import praw
import psycopg2

from postgres import create_connection, create_picture, get_latest
from s3_upload import init_s3
from scrape import get_pics, init_reddit

# Define subreddit names
ANALOG = "analog"
BW = "analog_bw"
SPROCKET = "SprocketShots"


@dataclasses.dataclass
class Resources:
    """
    Struct to hold common dependencies needed for scraper

    """

    conn: psycopg2.connect
    s3: boto3.session.Session
    reddit: praw.Reddit
    latest: List[str]


def setup_resources(test: bool):
    conn = create_connection(test)
    s3 = init_s3()
    reddit = init_reddit()
    latest = get_latest(conn)
    return Resources(conn, s3, reddit, latest)


def scrape_pics(r: Resources, subreddit: str, num_pics: int) -> None:
    for data in get_pics(r.reddit, r.s3, num_pics, subreddit, r.latest):
        if data.title not in r.latest:
            create_picture(r.conn, r.s3, dataclasses.astuple(data))


def test():
    test = True

    r = setup_resources(test)
    scrape_pics(r, subreddit=SPROCKET, num_pics=2)
    r.conn.close()


def main():
    test = False
    now = dt.datetime.now()

    # Scrape r/analog_bw and sprocketshots once a day
    if now.hour == 0:
        r = setup_resources(test)
        scrape_pics(r, subreddit=ANALOG, num_pics=7)
        scrape_pics(r, subreddit=BW, num_pics=2)
        scrape_pics(r, subreddit=SPROCKET, num_pics=1)
        r.conn.close()

    # Scrape r/analog every 8 hours
    elif now.hour == 8 or now.hour == 16:
        r = setup_resources()
        scrape_pics(r, subreddit=ANALOG, num_pics=7)


if __name__ == "__main__":
    # main()
    test()

import dataclasses
import datetime as dt

from postgres import create_connection, create_picture
from s3_upload import init_s3
from scrape import get_pics, init_reddit


def setup_resources():
    conn = create_connection(test)
    s3 = init_s3()
    reddit = init_reddit()
    return s3, conn, reddit


def scrape_analog(conn, s3, reddit):
    for data in get_pics(reddit, s3, num_pics=7, subreddit="analog"):
        create_picture(conn, s3, dataclasses.astuple(data))


def scrape_bw(conn, s3, reddit):
    for data in get_pics(reddit, s3, num_pics=2, subreddit="analog_bw"):
        create_picture(conn, s3, dataclasses.astuple(data))


def scrape_sprocket(conn, s3, reddit):
    for data in get_pics(reddit, s3, num_pics=1, subreddit="SprocketShots"):
        create_picture(conn, s3, dataclasses.astuple(data))


if __name__ == "__main__":

    test = False
    now = dt.datetime.now()

    # Scrape r/analog_bw and sprocketshots once a day
    if now.hour == 0:
        conn, s3, reddit = setup_resources()
        scrape_bw(conn, s3, reddit)
        scrape_sprocket(conn, s3, reddit)
        scrape_analog(conn, s3, reddit)
        conn.close()

    # Scrape r/analog every 8 hours
    elif now.hour == 8 or now.hour == 16:
        conn, s3, reddit = setup_resources()
        scrape_analog(conn, s3, reddit)
        conn.close()

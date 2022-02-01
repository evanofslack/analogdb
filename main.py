import dataclasses
import datetime as dt

from postgres import create_connection, create_picture, delete_post, update_all_urls
from s3_upload import init_s3
from scrape import get_pics


def scrape_analog(conn, s3):
    for data in get_pics(num_pics=7, subreddit="analog"):
        create_picture(conn, s3, dataclasses.astuple(data))


def scrape_bw(conn, s3):
    for data in get_pics(num_pics=2, subreddit="analog_bw"):
        create_picture(conn, s3, dataclasses.astuple(data))


def scrape_sprocket(conn, s3):
    for data in get_pics(num_pics=1, subreddit="SprocketShots"):
        create_picture(conn, s3, dataclasses.astuple(data))


if __name__ == "__main__":

    test = False
    now = dt.datetime.now()
    conn = create_connection(test)

    delete_posts = [782, 791, 787, 783, 790, 788, 789]
    for post in delete_posts:
        delete_post(conn, post)

    # # Scrape B&W and Sprocket once a day
    # if now.hour == 0:
    #     s3 = init_s3()
    #     conn = create_connection(test)
    #     scrape_bw(conn, s3)
    #     scrape_sprocket(conn, s3)
    #     scrape_analog(conn, s3)
    #     conn.close()

    # # Scrape r/analog every 8 hours
    # elif now.hour == 8 or now.hour == 16:
    #     s3 = init_s3()
    #     conn = create_connection(test)
    #     scrape_analog(conn, s3)
    #     conn.close()

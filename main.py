import dataclasses
import datetime as dt

from postgres import create_connection, create_picture, update_url
from scrape import get_pics


def scrape_analog(conn):
    for data in get_pics(num_pics=6, subreddit="analog"):
        create_picture(conn, dataclasses.astuple(data))


def scrape_bw(conn):
    for data in get_pics(num_pics=2, subreddit="analog_bw"):
        create_picture(conn, dataclasses.astuple(data))


def scrape_sprocket(conn):
    for data in get_pics(num_pics=1, subreddit="SprocketShots"):
        create_picture(conn, dataclasses.astuple(data))


if __name__ == "__main__":

    test = False
    now = dt.datetime.now()

    conn = create_connection(test)
    update_url(conn)
    conn.close()

    # # Scrape B&W and Sprocket once a day
    # if now.hour == 0:
    #     conn = create_connection(test)
    #     scrape_bw(conn)
    #     scrape_sprocket(conn)
    #     scrape_analog(conn)
    #     conn.close()

    # # Scrape r/analog every 8 hours
    # elif now.hour == 8 or now.hour == 16:
    #     conn = create_connection(test)
    #     scrape_analog(conn)
    #     conn.close()

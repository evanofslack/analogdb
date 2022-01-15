import dataclasses
import time

from postgres import create_connection, create_picture
from scrape import get_pics


def scrape_analog(conn):
    for data in get_pics(num_pics=3, subreddit="analog"):
        create_picture(conn, dataclasses.astuple(data))


def scrape_bw(conn):
    for data in get_pics(num_pics=2, subreddit="analog_bw"):
        create_picture(conn, dataclasses.astuple(data))


def scrape_sprocket(conn):
    for data in get_pics(num_pics=1, subreddit="SprocketShots"):
        create_picture(conn, dataclasses.astuple(data))


if __name__ == "__main__":
    test = False
    conn = create_connection(test)  # Create DB connection

    # scrape_bw(conn)  # Scrape top black & white picture once a day
    scrape_sprocket(conn)  # Scrape top sprocket shot once a day
    scrape_analog(conn)
    conn.close()

    # for i in range(3):  # Scrape top analog pictures approximately every 8 hours
    #     conn = create_connection(test)
    #     scrape_analog(conn)
    #     conn.close()
    #     time.sleep(60 * 60 * 8)  # Wait for 8 hours

    while True:
        # Heroku will restart container approximately every 24 hours
        time.sleep(60 * 60 * 24)

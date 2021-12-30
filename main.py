import dataclasses
import time

from postgres import create_connection, create_picture, create_table
from scrape import get_pics


def main():
    conn = create_connection(test=False)
    create_table(conn)

    for data in get_pics(num_pics=5):
        create_picture(conn, dataclasses.astuple(data))
    conn.close()


def get_all():
    conn = create_connection(test=True)
    c = conn.cursor()
    c.execute("SELECT * FROM pictures")
    row = c.fetchone()

    while row is not None:
        print(row)
        row = c.fetchone()


if __name__ == "__main__":

    while True:
        main()

        # Heroku schedular limited to once a day
        # Ideally we want to run twice a day
        # Timing is not critical though.
        time.sleep(60 * 60 * 12)  # Wait for 12 hours

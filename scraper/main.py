import dataclasses
import os

from postgres import create_connection, create_picture, create_table
from scrape import get_pics

db_file = os.path.dirname(os.getcwd()) + "/test.db"
create_picture_table = """CREATE TABLE IF NOT EXISTS pictures (
                            id SERIAL PRIMARY KEY, 
                            url text NOT NULL, 
                            title text, 
                            permalink text,
                            score integer,
                            nsfw boolean,
                            time text
                            );"""


def main():
    conn = create_connection()
    create_table(conn, create_picture_table)

    for data in get_pics():
        create_picture(conn, dataclasses.astuple(data))
    conn.close()


if __name__ == "__main__":
    # main()

    conn = create_connection()
    c = conn.cursor()
    c.execute("SELECT * FROM pictures")
    row = c.fetchone()

    while row is not None:
        print(row)
        row = c.fetchone()

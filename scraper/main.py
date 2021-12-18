import dataclasses
import os

from database import create_connection, create_picture, create_table
from scrape import get_pics

db_file = os.path.dirname(os.getcwd()) + "/test.db"
create_picture_table = """CREATE TABLE IF NOT EXISTS pictures (
                            id integer PRIMARY KEY, 
                            url text NOT NULL, 
                            title text, 
                            permalink text,
                            score integer,
                            nsfw integer,
                            time text
                            );"""


def main():
    conn = create_connection(db_file)
    create_table(conn, create_picture_table)

    for data in get_pics():
        create_picture(conn, dataclasses.astuple(data))
    conn.close()


if __name__ == "__main__":
    main()

    conn = create_connection(db_file)
    c = conn.cursor()
    for row in c.execute("SELECT * FROM pictures"):
        print(row)

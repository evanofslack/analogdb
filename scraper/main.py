import os

from database import create_connection, create_picture, create_table
from scrape import get_pics

db_file = os.path.dirname(os.getcwd()) + "/test.db"
create_picture_table = "CREATE TABLE IF NOT EXISTS pictures (id integer PRIMARY KEY, url TEXT NOT NULL, raw TEXT NOT NULL)"


def main():
    conn = create_connection(db_file)
    create_table(conn, create_picture_table)

    pic_data = get_pics()

    for data in pic_data:
        create_picture(conn, data)

    c = conn.cursor()
    for row in c.execute("SELECT * FROM pictures"):
        print(row)

    conn.close()


if __name__ == "__main__":
    main()

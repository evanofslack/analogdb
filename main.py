import dataclasses

from postgres import create_connection, create_picture, create_table
from scrape import get_pics


def main():
    conn = create_connection(test=True)
    create_table(conn)

    for data in get_pics():
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
    main()

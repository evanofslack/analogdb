from database import create_connection, create_picture, create_table
from scrape import get_pics

create_picture_table = "CREATE TABLE IF NOT EXISTS pictures (id integer PRIMARY KEY, url TEXT NOT NULL, raw TEXT NOT NULL)"
db_file = "test.db"


def main():
    conn = create_connection(db_file)
    if conn is not None:
        create_table(conn, create_picture_table)
    else:
        print("Could not connect to database")

    pic_data = get_pics()
    # print(pic_data)

    for data in pic_data:
        create_picture(conn, data)

    c = conn.cursor()
    for row in c.execute("SELECT * FROM pictures"):
        print(row)

    conn.close()


if __name__ == "__main__":
    main()

import sqlite3


def create_connection(db_file: str) -> sqlite3.Connection:
    connection = None
    try:
        connection = sqlite3.connect(db_file)
    except sqlite3.Error as e:
        print(e)

    return connection


def create_table(connection: sqlite3.Connection, create_table: str):
    try:
        c = connection.cursor()
        c.execute(create_table)
    except sqlite3.Error as e:
        print(e)


def drop_table(connection: sqlite3.Connection, table: str):
    try:
        c = connection.cursor()
        c.execute("DROP table ?", table)
    except sqlite3.Error as e:
        print(e)


def create_picture(conn: sqlite3.Connection, data: tuple):

    try:
        c = conn.cursor()
        c.execute(
            "INSERT INTO pictures(title, url, permalink, score, nsfw, time) VALUES (?,?,?,?,?,?)",
            data,
        )
        conn.commit()
    except sqlite3.Error as e:
        print(e)

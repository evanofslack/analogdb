import os

import psycopg2


def create_connection(test: bool = False):
    connection = None
    try:
        if not test:
            connection = psycopg2.connect(os.environ.get("DATABASE_URL"))
        elif test:
            connection = psycopg2.connect(
                host=os.environ.get("DBHOST"),
                database=os.environ.get("DBNAME"),
                user=os.environ.get("DBUSER"),
                password=os.environ.get("DBPASSWORD"),
            )
        else:
            raise Exception("Must set database init")
        return connection
    except (Exception, psycopg2.DatabaseError) as error:
        print(error)


def create_table(connection):

    create_picture_table = """CREATE TABLE IF NOT EXISTS pictures (
                            id SERIAL PRIMARY KEY, 
                            url text NOT NULL UNIQUE, 
                            title text, 
                            author text,
                            permalink text,
                            score integer,
                            nsfw boolean,
                            greyscale boolean,
                            time integer,
                            width integer,
                            height integer
                            );"""
    try:
        c = connection.cursor()
        c.execute(create_picture_table)
        connection.commit()
    except (Exception, psycopg2.DatabaseError) as error:
        print(error)


def create_picture(conn, data: tuple):
    try:
        c = conn.cursor()
        c.execute(
            """
            INSERT 
            INTO pictures(url, title, author, permalink, score, nsfw, greyscale, time, width, height) 
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s) 
            ON CONFLICT (url) DO NOTHING
            """,
            data,
        )
        conn.commit()
    except (Exception, psycopg2.DatabaseError) as error:
        print(error)

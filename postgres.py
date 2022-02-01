import os

import psycopg2

from s3_upload import UploadError, init_s3, s3_upload


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


def create_picture(conn, s3, data: tuple):
    try:
        c = conn.cursor()
        c.execute(
            """
            INSERT 
            INTO pictures(url, title, author, permalink, score, nsfw, greyscale, time, width, height, sprocket) 
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s) 
            ON CONFLICT (permalink) DO NOTHING
            RETURNING id,url
            """,
            data,
        )
        result = c.fetchone()
        id = result[0]
        url = result[1]
        update_url(conn, s3, id, url)
        conn.commit()
    except (Exception, psycopg2.DatabaseError) as error:
        print(error)


def update_table(conn):
    try:
        c = conn.cursor()
        c.execute(
            """
            ALTER TABLE pictures
            ADD COLUMN sprocket BOOLEAN DEFAULT FALSE
            """,
        )
        conn.commit()
    except (Exception, psycopg2.DatabaseError) as error:
        print(error)


def get_tables(conn):
    cursor = conn.cursor()
    cursor.execute(
        """SELECT table_name FROM information_schema.tables
       WHERE table_schema = 'public'"""
    )
    for table in cursor.fetchall():
        print(table)


def get_columns(conn):
    c = conn.cursor()
    c.execute("Select * FROM pictures LIMIT 0")
    colnames = [desc[0] for desc in c.description]
    print(colnames)


def get_all(conn):
    c = conn.cursor()
    c.execute("""SELECT * FROM pictures""")
    row = c.fetchone()

    while row is not None:
        print(row)
        row = c.fetchone()


def delete_post(conn, post: int):
    try:
        c = conn.cursor()
        c.execute("""DELETE FROM pictures WHERE id = (%s)""", (post,))
        conn.commit()
        print(f"deleted {post}")

    except (Exception, psycopg2.DatabaseError) as error:
        print(f"db_error: {error}")


def update_url(conn, s3, id, url):
    query = """ UPDATE pictures
                SET url = %s
                WHERE id = %s"""

    c = conn.cursor()
    new_url = s3_upload(s3, bucket="analog-photos", url=url, filename=id)
    c.execute(query, (new_url, id))


def update_all_urls(conn):
    """
    Upload reddit images to S3 and update db URL with Cloudfront URL

    """
    query = """ UPDATE pictures
                SET url = %s
                WHERE id = %s"""

    s3 = init_s3()

    c = conn.cursor()
    c.execute("""SELECT id, url FROM pictures""")
    row = c.fetchone()

    new_rows = []

    while row is not None:
        id = str(row[0])
        url = row[1]

        try:
            new_url = s3_upload(s3, bucket="analog-photos", url=url, filename=id)
            new_rows.append((new_url, id))
        except UploadError:
            pass

        row = c.fetchone()

    for row in new_rows:
        c.execute(query, (row[0], row[1]))

    conn.commit()


if __name__ == "__main__":

    conn = create_connection(True)
    get_all(conn)

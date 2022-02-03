import psycopg2

from s3_upload import UploadError, s3_upload


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


def delete_post(conn, post: int):
    try:
        c = conn.cursor()
        c.execute("""DELETE FROM pictures WHERE id = (%s)""", (post,))
        conn.commit()
        print(f"deleted {post}")

    except (Exception, psycopg2.DatabaseError) as error:
        print(f"db_error: {error}")


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


def alter_table(conn):
    query = """ALTER TABLE pictures
            ADD CONSTRAINT unique_url 
            UNIQUE (permalink);"""
    try:
        c = conn.cursor()
        c.execute(query)
        conn.commit()
        print("Success, altered table")
    except (Exception, psycopg2.DatabaseError) as error:
        print(error)

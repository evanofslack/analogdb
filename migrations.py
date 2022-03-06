import psycopg2

from postgres import create_connection
from s3_upload import UploadError, init_s3, s3_upload
from scrape import url_to_images


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


def resize_all_photos(conn):
    """
    Resize and upload images to S3 and populate DB with Cloudfront URLs

    """
    query = """ UPDATE pictures
                SET url = %s,
                    width = %s,
                    height = %s,
                    lowUrl = %s,
                    lowWidth = %s,
                    lowHeight = %s,
                    medUrl = %s,
                    medWidth = %s,
                    medHeight = %s,
                    highUrl = %s,
                    highWidth = %s,
                    highHeight = %s

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
            images = url_to_images(url, s3, bucket="analog-photos")
            low = images[0]
            med = images[1]
            high = images[2]
            raw = images[3]

            new_rows.append(
                (
                    raw.url,
                    raw.width,
                    raw.height,
                    low.url,
                    low.width,
                    low.height,
                    med.url,
                    med.width,
                    med.height,
                    high.url,
                    high.width,
                    high.height,
                    id,
                )
            )
        except Exception as e:
            print(f"Error resizing/uploading post: {id}, error: {e}")
            pass

        row = c.fetchone()

    for row in new_rows:
        c.execute(
            query,
            (
                row[0],
                row[1],
                row[2],
                row[3],
                row[4],
                row[5],
                row[6],
                row[7],
                row[8],
                row[9],
                row[10],
                row[11],
                row[12],
            ),
        )

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
    c = conn.cursor()
    c.execute(
        """
        ALTER TABLE pictures
        ADD COLUMN lowUrl text, 
        ADD COLUMN lowWidth integer, 
        ADD COLUMN lowHeight integer, 
        ADD COLUMN medUrl text, 
        ADD COLUMN medWidth integer, 
        ADD COLUMN medHeight integer, 
        ADD COLUMN highUrl text, 
        ADD COLUMN highWidth integer, 
        ADD COLUMN highHeight integer 
        """,
    )
    conn.commit()
    print("Success, updated table")


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


if __name__ == "__main__":
    conn = create_connection(test=True)
    # update_table(conn)
    resize_all_photos(conn)

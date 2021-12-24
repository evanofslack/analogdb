import psycopg2


def create_connection():
    connection = None
    try:
        connection = psycopg2.connect(
            host="localhost", database="Test", user="evan", password="dark"
        )
        return connection
    except (Exception, psycopg2.DatabaseError) as error:
        print(error)


def create_table(connection, command: str):
    try:
        c = connection.cursor()
        c.execute(command)
        connection.commit()
    except (Exception, psycopg2.DatabaseError) as error:
        print(error)


def create_picture(conn, data: tuple):
    try:
        c = conn.cursor()
        c.execute(
            "INSERT INTO pictures(title, url, permalink, score, nsfw, time) VALUES (%s, %s, %s, %s, %s, %s)",
            data,
        )
        conn.commit()
    except (Exception, psycopg2.DatabaseError) as error:
        print(error)

import sqlite3

connection = sqlite3.connect("test.db")

cursor = connection.cursor()
cursor.execute(
    "CREATE TABLE IF NOT EXISTS pictures (id integer PRIMARY KEY, url TEXT NOT NULL)"
)

cursor.execute("INSERT INTO pictures VALUES (1, 'www.example.com')")

rows = cursor.execute("SELECT id, url FROM pictures").fetchall()
print(rows)

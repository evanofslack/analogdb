package models

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

var db *sql.DB

func InitDB() error {
	test := false
	var psqlInfo string
	if test {
		psqlInfo = fmt.Sprintf("host=%s port=%s user=%s "+
			"password=%s dbname=%s sslmode=disable",
			os.Getenv("DBHOST"), os.Getenv("DBPORT"), os.Getenv("DBUSER"),
			os.Getenv("DBPASSWORD"), os.Getenv("DBNAME"))
	} else {
		conn, _ := pq.ParseURL(os.Getenv("DATABASE_URL"))
		psqlInfo = conn + "sslmode=require"

	}

	var err error

	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		return err
	}
	return db.Ping()
}

type Post struct {
	id        int
	Url       string
	title     string
	author    string
	permalink string
	score     int
	nsfw      bool
	greyscale bool
	time      string
	width     int
	height    int
}

func AllPosts() ([]Post, error) {

	rows, err := db.Query("SELECT * FROM pictures")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post

	for rows.Next() {
		var p Post
		err := rows.Scan(&p.id, &p.Url, &p.title, &p.author, &p.permalink, &p.score, &p.nsfw, &p.greyscale, &p.time, &p.width, &p.height)
		if err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	for _, post := range posts {
		fmt.Println(post)
	}

	return posts, nil
}

func LatestPost() (*Post, error) {

	rows, err := db.Query("SELECT * FROM pictures ORDER BY id DESC LIMIT 1;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var p Post

	for rows.Next() {
		err := rows.Scan(&p.id, &p.Url, &p.title, &p.author, &p.permalink, &p.score, &p.nsfw, &p.greyscale, &p.time, &p.width, &p.height)
		if err != nil {
			return nil, err
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	fmt.Println(p)
	return &p, nil
}

func RandomPost() (*Post, error) {

	rows, err := db.Query("SELECT * FROM pictures ORDER BY RANDOM() LIMIT 1;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var p Post

	for rows.Next() {
		err := rows.Scan(&p.id, &p.Url, &p.title, &p.author, &p.permalink, &p.score, &p.nsfw, &p.greyscale, &p.time, &p.width, &p.height)
		if err != nil {
			return nil, err
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &p, nil
}

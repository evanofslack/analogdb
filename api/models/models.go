package models

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func InitDB(path string) error {
	var err error

	db, err = sql.Open("sqlite3", path)
	if err != nil {
		return err
	}
	return db.Ping()
}

type Post struct {
	id        int
	url       string
	title     string
	permalink string
	score     int
	nsfw      int
	time      string
}

func AllPosts() ([]Post, error) {

	rows, err := db.Query("SELECT id, url, title, permalink, score, nsfw, time from pictures")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post

	for rows.Next() {
		var p Post
		err := rows.Scan(&p.id, &p.url, &p.title, &p.permalink, &p.score, &p.nsfw, &p.time)
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

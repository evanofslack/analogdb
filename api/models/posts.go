package models

import "fmt"

type Post struct {
	id        int
	Url       string
	Title     string
	Author    string
	Permalink string
	Score     int
	Nsfw      bool
	Greyscale bool
	Time      string
	Width     int
	Height    int
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
		err := rows.Scan(&p.id, &p.Url, &p.Title, &p.Author, &p.Permalink, &p.Score, &p.Nsfw, &p.Greyscale, &p.Time, &p.Width, &p.Height)
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
		err := rows.Scan(&p.id, &p.Url, &p.Title, &p.Author, &p.Permalink, &p.Score, &p.Nsfw, &p.Greyscale, &p.Time, &p.Width, &p.Height)
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
		err := rows.Scan(&p.id, &p.Url, &p.Title, &p.Author, &p.Permalink, &p.Score, &p.Nsfw, &p.Greyscale, &p.Time, &p.Width, &p.Height)
		if err != nil {
			return nil, err
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &p, nil
}

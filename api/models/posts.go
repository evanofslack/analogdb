package models

import (
	"database/sql"
	"fmt"
	"strconv"
)

type Post struct {
	id        int
	Url       string `json:"url"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	Permalink string `json:"permalink"`
	Score     int    `json:"upvotes"`
	Nsfw      bool   `json:"nsfw"`
	Greyscale bool   `json:"greyscale"`
	Time      int    `json:"unix_time"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type Response struct {
	PageID string `json:"next_page_id"`
	Posts  []Post `json:"posts"`
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

func LatestPost(limit int, time int) (Response, error) {

	var rows *sql.Rows
	var err error

	if time == 0 {
		rows, err = db.Query("SELECT * FROM pictures ORDER BY time DESC LIMIT $1;", limit)
	} else {
		rows, err = db.Query("SELECT * FROM pictures WHERE time < $1 ORDER BY time DESC LIMIT $2;", time, limit)
	}
	if err != nil {
		return Response{}, err
	}
	defer rows.Close()

	var response Response

	for rows.Next() {
		var p Post
		err := rows.Scan(&p.id, &p.Url, &p.Title, &p.Author, &p.Permalink, &p.Score, &p.Nsfw, &p.Greyscale, &p.Time, &p.Width, &p.Height)
		if err != nil {
			return Response{}, err
		}
		response.Posts = append(response.Posts, p)
	}

	if err = rows.Err(); err != nil {
		return Response{}, err
	}
	if len(response.Posts) == limit {
		response.PageID = strconv.Itoa(response.Posts[limit-1].Time)
	} else {
		response.PageID = ""
	}
	return response, nil
}

func TopPost(num int) (Response, error) {

	rows, err := db.Query("SELECT * FROM pictures ORDER BY score DESC LIMIT $1;", num)
	if err != nil {
		return Response{}, err
	}
	defer rows.Close()

	var response Response

	for rows.Next() {
		var p Post
		err := rows.Scan(&p.id, &p.Url, &p.Title, &p.Author, &p.Permalink, &p.Score, &p.Nsfw, &p.Greyscale, &p.Time, &p.Width, &p.Height)
		if err != nil {
			return Response{}, err
		}
		response.Posts = append(response.Posts, p)
	}

	if err = rows.Err(); err != nil {
		return Response{}, err
	}
	return response, nil
}
func RandomPost(num int) (Response, error) {

	rows, err := db.Query("SELECT * FROM pictures ORDER BY RANDOM() LIMIT $1;", num)
	if err != nil {
		return Response{}, err
	}
	defer rows.Close()

	var response Response

	for rows.Next() {
		var p Post
		err := rows.Scan(&p.id, &p.Url, &p.Title, &p.Author, &p.Permalink, &p.Score, &p.Nsfw, &p.Greyscale, &p.Time, &p.Width, &p.Height)
		if err != nil {
			return Response{}, err
		}
		response.Posts = append(response.Posts, p)
	}

	if err = rows.Err(); err != nil {
		return Response{}, err
	}

	return response, nil
}

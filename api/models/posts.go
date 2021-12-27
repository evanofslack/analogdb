package models

import "fmt"

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
	Posts []Post `json:"posts"`
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

func LatestPost(num int) (Response, error) {

	rows, err := db.Query("SELECT * FROM pictures ORDER BY time DESC LIMIT $1;", num)
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

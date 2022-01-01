package models

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strconv"
)

type Post struct {
	Id        int    `json:"id"`
	Url       string `json:"url"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	Permalink string `json:"permalink"`
	Score     int    `json:"upvotes"`
	Nsfw      bool   `json:"nsfw"`
	Grayscale bool   `json:"grayscale"`
	Time      int    `json:"unix_time"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type Meta struct {
	TotalPosts int    `json:"total_posts"`
	PageSize   int    `json:"page_size"`
	PageID     string `json:"next_page_id"`
	PageURL    string `json:"next_page_url"`
	Seed       int    `json:"seed,omitempty"`
}

type Response struct {
	Meta  Meta   `json:"meta"`
	Posts []Post `json:"posts"`
}

func LatestPost(limit int, time int) (Response, error) {

	var rows *sql.Rows
	var err error
	var response Response

	if time == 0 {
		rows, err = db.Query("SELECT * FROM pictures ORDER BY time DESC LIMIT $1;", limit)
	} else {
		rows, err = db.Query("SELECT * FROM pictures WHERE time < $1 ORDER BY time DESC LIMIT $2;", time, limit)
	}
	if err != nil {
		return Response{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var p Post
		err := rows.Scan(&p.Id, &p.Url, &p.Title, &p.Author, &p.Permalink, &p.Score, &p.Nsfw, &p.Grayscale, &p.Time, &p.Width, &p.Height)
		if err != nil {
			return Response{}, err
		}
		response.Posts = append(response.Posts, p)
	}

	if err = rows.Err(); err != nil {
		return Response{}, err
	}

	// Set response metadata
	response.Meta.TotalPosts = getRowCount()
	response.Meta.PageSize = limit
	if len(response.Posts) == limit {
		pageID := strconv.Itoa(response.Posts[limit-1].Time)
		response.Meta.PageID = pageID
		response.Meta.PageURL = "/latest?page_size=" + strconv.Itoa(limit) + "&page_id=" + pageID
	} else {
		response.Meta.PageID = ""
		response.Meta.PageURL = ""
	}
	return response, nil
}

func TopPost(limit int, score int) (Response, error) {

	var rows *sql.Rows
	var err error
	var response Response

	if score == 0 {
		rows, err = db.Query("SELECT * FROM pictures ORDER BY score DESC LIMIT $1;", limit)
	} else {
		rows, err = db.Query("SELECT * FROM pictures WHERE score < $1 ORDER BY time DESC LIMIT $2;", score, limit)
	}
	if err != nil {
		return Response{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var p Post
		err := rows.Scan(&p.Id, &p.Url, &p.Title, &p.Author, &p.Permalink, &p.Score, &p.Nsfw, &p.Grayscale, &p.Time, &p.Width, &p.Height)
		if err != nil {
			return Response{}, err
		}
		response.Posts = append(response.Posts, p)
	}

	// Set response metadata
	response.Meta.TotalPosts = getRowCount()
	response.Meta.PageSize = limit
	if err = rows.Err(); err != nil {
		return Response{}, err
	}
	if len(response.Posts) == limit {
		pageID := strconv.Itoa(response.Posts[limit-1].Score)
		response.Meta.PageID = pageID
		response.Meta.PageURL = "/top?page_size=" + strconv.Itoa(limit) + "&page_id=" + pageID
	} else {
		response.Meta.PageID = ""
		response.Meta.PageURL = ""
	}
	return response, nil
}
func RandomPost(limit int, time int, seed int) (Response, error) {

	if seed == 0 {
		prime_seeds := []int{11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71, 73, 79, 83, 89, 97}
		randomIndex := rand.Intn(len(prime_seeds))
		seed = prime_seeds[randomIndex]
	}

	var rows *sql.Rows
	var err error
	var response Response

	// Create shuffled order of db based on seed to create "random" order that is repeatable if the seed is supplied.
	if time == 0 {
		rows, err = db.Query("SELECT * FROM pictures ORDER BY time % $1, time DESC LIMIT $2;", seed, limit)
	} else {
		rows, err = db.Query("SELECT * FROM pictures WHERE time % $1 > $2 ORDER BY time % $3, time DESC LIMIT $4;", seed, time%seed, seed, limit)
	}

	if err != nil {
		return Response{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var p Post
		err := rows.Scan(&p.Id, &p.Url, &p.Title, &p.Author, &p.Permalink, &p.Score, &p.Nsfw, &p.Grayscale, &p.Time, &p.Width, &p.Height)
		if err != nil {
			return Response{}, err
		}
		response.Posts = append(response.Posts, p)
	}

	if err = rows.Err(); err != nil {
		return Response{}, err
	}

	// Set response metadata
	response.Meta.TotalPosts = getRowCount()
	response.Meta.PageSize = limit
	if len(response.Posts) == limit {
		pageID := strconv.Itoa(response.Posts[limit-1].Time)
		strSeed := strconv.Itoa(seed)
		response.Meta.PageID = pageID
		response.Meta.PageURL = "/random?page_size=" + strconv.Itoa(limit) + "&page_id=" + pageID + "&seed=" + strSeed
	} else {
		response.Meta.PageID = ""
		response.Meta.PageURL = ""
	}
	response.Meta.Seed = seed

	return response, nil
}

func getRowCount() int {
	rows, err := db.Query("SELECT COUNT(*) as count FROM  pictures")
	if err != nil {
		fmt.Println(err)
		return 0
	}
	var count int
	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			fmt.Println(err)
			return 0
		}
	}
	return count
}

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

func LatestPost(limit int, time int, nsfw bool, grayscale bool) (Response, error) {

	fmt.Println(nsfw, grayscale)
	var rows *sql.Rows
	var err error
	var response Response
	var statement string

	if time == 0 {
		if !nsfw && !grayscale {
			statement = "SELECT * FROM pictures WHERE greyscale = FALSE and nsfw = FALSE ORDER BY time DESC LIMIT $1;"
		} else if !nsfw {
			statement = "SELECT * FROM pictures WHERE nsfw = FALSE ORDER BY time DESC LIMIT $1;"
		} else if !grayscale {
			statement = "SELECT * FROM pictures WHERE greyscale = FALSE ORDER BY time DESC LIMIT $1;"
		} else {
			statement = "SELECT * FROM pictures ORDER BY time DESC LIMIT $1;"
		}
		rows, err = db.Query(statement, limit)
	} else {
		if !nsfw && !grayscale {
			statement = "SELECT * FROM pictures WHERE time < $1 and greyscale = FALSE and nsfw = FALSE ORDER BY time DESC LIMIT $2;"
		} else if !nsfw {
			statement = "SELECT * FROM pictures WHERE time < $1 and nsfw = FALSE ORDER BY time DESC LIMIT $2;"
		} else if !grayscale {
			statement = "SELECT * FROM pictures WHERE time < $1 and greyscale = FALSE ORDER BY time DESC LIMIT $2;"
		} else {
			statement = "SELECT * FROM pictures WHERE time < $1 ORDER BY time DESC LIMIT $2;"
		}
		rows, err = db.Query(statement, time, limit)
	}
	if err != nil {
		return Response{}, err
	}

	response, err = createResponse(rows)
	if err != nil {
		return Response{}, err
	}

	// Set response metadata
	response.Meta.TotalPosts = getRowCount(nsfw, grayscale)
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
	response.Meta.TotalPosts = getRowCount(true, true)
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
	response.Meta.TotalPosts = getRowCount(false, true)
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

func getRowCount(nsfw bool, grayscale bool) int {
	var statement string
	if !nsfw && !grayscale {
		statement = "SELECT COUNT(*) as count FROM pictures WHERE nsfw = FALSE and greyscale = FALSE"
	} else if !nsfw {
		statement = "SELECT COUNT(*) as count FROM pictures WHERE nsfw = FALSE"
	} else if !grayscale {
		statement = "SELECT COUNT(*) as count FROM pictures WHERE greyscale = FALSE"
	} else {
		statement = "SELECT COUNT(*) as count FROM pictures"
	}
	rows, err := db.Query(statement)
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

// Turn rows from db query into response struct
func createResponse(rows *sql.Rows) (Response, error) {
	var response Response
	var err error
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
	return response, nil
}

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
	Sprocket  bool   `json:"sprocket"`
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
	basePath := "/latest"
	response = setMeta(response, limit, basePath, nsfw, grayscale, false, false, "time")
	return response, nil
}

func TopPost(limit int, score int, nsfw bool, grayscale bool) (Response, error) {

	var rows *sql.Rows
	var err error
	var response Response
	var statement string

	if score == 0 {
		if !nsfw && !grayscale {
			statement = "SELECT * FROM pictures WHERE greyscale = FALSE and nsfw = FALSE ORDER BY score DESC LIMIT $1;"
		} else if !nsfw {
			statement = "SELECT * FROM pictures WHERE nsfw = FALSE ORDER BY score DESC LIMIT $1;"
		} else if !grayscale {
			statement = "SELECT * FROM pictures WHERE greyscale = FALSE ORDER BY score DESC LIMIT $1;"
		} else {
			statement = "SELECT * FROM pictures ORDER BY score DESC LIMIT $1;"
		}
		rows, err = db.Query(statement, limit)
	} else {
		if !nsfw && !grayscale {
			statement = "SELECT * FROM pictures WHERE score < $1 and greyscale = FALSE and nsfw = FALSE ORDER BY score DESC LIMIT $2;"
		} else if !nsfw {
			statement = "SELECT * FROM pictures WHERE score < $1 and nsfw = FALSE ORDER BY score DESC LIMIT $2;"
		} else if !grayscale {
			statement = "SELECT * FROM pictures WHERE score < $1 and greyscale = FALSE ORDER BY score DESC LIMIT $2;"
		} else {
			statement = "SELECT * FROM pictures WHERE score < $1 ORDER BY score DESC LIMIT $2;"
		}
		rows, err = db.Query(statement, score, limit)
	}
	if err != nil {
		return Response{}, err
	}
	response, err = createResponse(rows)
	if err != nil {
		return Response{}, err
	}
	basePath := "/top"
	response = setMeta(response, limit, basePath, nsfw, grayscale, false, false, "score")
	return response, nil
}

func RandomPost(limit int, time int, nsfw bool, grayscale bool, seed int) (Response, error) {
	if seed == 0 {
		prime_seeds := []int{11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71, 73, 79, 83, 89, 97}
		randomIndex := rand.Intn(len(prime_seeds))
		seed = prime_seeds[randomIndex]
	}

	var rows *sql.Rows
	var err error
	var response Response
	var statement string

	// Create shuffled order of db based on seed to create "random" order that is repeatable if the seed is supplied.
	if time == 0 {
		if !nsfw && !grayscale {
			statement = "SELECT * FROM pictures WHERE nsfw = FALSE and greyscale = FALSE ORDER BY time % $1, time DESC LIMIT $2;"
		} else if !nsfw {
			statement = "SELECT * FROM pictures WHERE nsfw = FALSE ORDER BY time % $1, time DESC LIMIT $2;"
		} else if !grayscale {
			statement = "SELECT * FROM pictures WHERE greyscale = FALSE ORDER BY time % $1, time DESC LIMIT $2;"
		} else {
			statement = "SELECT * FROM pictures ORDER BY time % $1, time DESC LIMIT $2;"
		}
		rows, err = db.Query(statement, seed, limit)
	} else {
		if !nsfw && !grayscale {
			statement = "SELECT * FROM pictures WHERE time % $1 > $2 and nsfw = FALSE and greyscale = FALSE ORDER BY time % $3, time DESC LIMIT $4;"
		} else if !nsfw {
			statement = "SELECT * FROM pictures WHERE time % $1 > $2 and nsfw = FALSE ORDER BY time % $3, time DESC LIMIT $4;"
		} else if !grayscale {
			statement = "SELECT * FROM pictures WHERE time % $1 > $2 and greyscale = FALSE ORDER BY time % $3, time DESC LIMIT $4;"
		} else {
			statement = "SELECT * FROM pictures WHERE time % $1 > $2 ORDER BY time % $3, time DESC LIMIT $4;"
		}
		rows, err = db.Query(statement, seed, (time % seed), seed, limit)
	}

	if err != nil {
		return Response{}, err
	}
	response, err = createResponse(rows)
	if err != nil {
		return Response{}, err
	}
	basePath := "/random"
	response = setMeta(response, limit, basePath, nsfw, grayscale, false, false, "time")
	response.Meta.PageURL += "&seed=" + strconv.Itoa(seed)
	response.Meta.Seed = seed

	return response, nil
}

func NsfwPost(limit int, time int) (Response, error) {

	var rows *sql.Rows
	var err error
	var response Response
	var statement string

	if time == 0 {
		statement = "SELECT * FROM pictures WHERE nsfw = TRUE ORDER BY time DESC LIMIT $1;"
		rows, err = db.Query(statement, limit)
	} else {
		statement = "SELECT * FROM pictures WHERE time < $1 and nsfw = TRUE ORDER BY time DESC LIMIT $2;"
		rows, err = db.Query(statement, time, limit)
	}
	if err != nil {
		return Response{}, err
	}
	response, err = createResponse(rows)
	if err != nil {
		return Response{}, err
	}
	basePath := "/nsfw"
	response = setMeta(response, limit, basePath, false, false, true, false, "time")
	return response, nil
}

func BwPost(limit int, time int) (Response, error) {

	var rows *sql.Rows
	var err error
	var response Response
	var statement string

	if time == 0 {
		statement = "SELECT * FROM pictures WHERE greyscale = TRUE ORDER BY time DESC LIMIT $1;"
		rows, err = db.Query(statement, limit)
	} else {
		statement = "SELECT * FROM pictures WHERE time < $1 and greyscale = TRUE ORDER BY time DESC LIMIT $2;"
		rows, err = db.Query(statement, time, limit)
	}
	if err != nil {
		return Response{}, err
	}
	response, err = createResponse(rows)
	if err != nil {
		return Response{}, err
	}
	basePath := "/bw"
	response = setMeta(response, limit, basePath, false, false, false, true, "time")
	return response, nil
}

// FindPost find and return post by ID
func FindPost(id int) (Post, error) {
	var p Post
	rows, err := db.Query("SELECT * FROM pictures WHERE id = $1;", id)
	if err != nil {
		fmt.Println(err)
		return Post{}, err
	}
	for rows.Next() {
		err := rows.Scan(&p.Id, &p.Url, &p.Title, &p.Author, &p.Permalink, &p.Score, &p.Nsfw, &p.Grayscale, &p.Time, &p.Width, &p.Height)
		if err != nil {
			return Post{}, err
		}
	}
	if err = rows.Err(); err != nil {
		return Post{}, err
	}
	return p, nil
}

// Get total number of entries in table for query
func getRowCount(nsfw bool, grayscale bool, onlyNsfw bool, onlyBw bool) int {
	var statement string
	if onlyNsfw {
		statement = "SELECT COUNT(*) as count FROM pictures WHERE nsfw = TRUE"
	} else if onlyBw {
		statement = "SELECT COUNT(*) as count FROM pictures WHERE greyscale = TRUE"
	} else if !nsfw && !grayscale {
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
		err := rows.Scan(&p.Id, &p.Url, &p.Title, &p.Author, &p.Permalink, &p.Score, &p.Nsfw, &p.Grayscale, &p.Time, &p.Width, &p.Height, &p.Sprocket)
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

// Set response metadata
func setMeta(r Response, limit int, basePath string, nsfw bool, grayscale bool, onlyNsfw bool, onlyBw bool, offsetKey string) Response {

	var pageID string

	r.Meta.TotalPosts = getRowCount(nsfw, grayscale, onlyNsfw, onlyBw)
	r.Meta.PageSize = limit

	if len(r.Posts) == limit {
		if offsetKey == "time" {
			pageID = strconv.Itoa(r.Posts[limit-1].Time)
		} else if offsetKey == "score" {
			pageID = strconv.Itoa(r.Posts[limit-1].Score)
		}
		r.Meta.PageID = pageID
		r.Meta.PageURL = basePath + "?page_size=" + strconv.Itoa(limit) + "&page_id=" + pageID
		if onlyNsfw || onlyBw {
			return r
		}
		if nsfw {
			r.Meta.PageURL += "&nsfw=true"
		}
		if !grayscale {
			r.Meta.PageURL += "&bw=false"
		}
	} else {
		r.Meta.PageID = ""
		r.Meta.PageURL = ""
	}
	return r
}

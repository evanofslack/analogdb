package models

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/lib/pq"
)

var db *sql.DB

func InitDB() error {
	LoadEnv()

	// psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
	// 	"password=%s dbname=%s sslmode=disable",
	// 	os.Getenv("DBHOST"), os.Getenv("DBPORT"), os.Getenv("DBUSER"),
	// 	os.Getenv("DBPASSWORD"), os.Getenv("DBNAME"))

	conn, _ := pq.ParseURL(os.Getenv("DATABASE_URL"))
	psqlInfo := conn + "sslmode=require"

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
	permalink string
	score     int
	nsfw      bool
	time      string
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
		err := rows.Scan(&p.id, &p.Url, &p.title, &p.permalink, &p.score, &p.nsfw, &p.time)
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

	// p := Post{}
	var p Post

	for rows.Next() {
		err := rows.Scan(&p.id, &p.Url, &p.title, &p.permalink, &p.score, &p.nsfw, &p.time)
		if err != nil {
			return nil, err
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

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
		err := rows.Scan(&p.id, &p.Url, &p.title, &p.permalink, &p.score, &p.nsfw, &p.time)
		if err != nil {
			return nil, err
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &p, nil
}

func LoadEnv() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal(err)
	}
}

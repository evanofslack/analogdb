package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type analog struct {
	id        int
	url       string
	title     string
	permalink string
	score     int
	nsfw      int
	time      string
}

func main() {

	db, err := sql.Open("sqlite3", parent_dir()+"/test.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err == nil {
		fmt.Println("Connection Established")
	} else {
		log.Fatal(err)
	}

	// rows, err := db.Query("SELECT id, url, title, permalink, score, nsfw, time from pictures WHERE id = ?", 1)
	rows, err := db.Query("SELECT id, url, title, permalink, score, nsfw, time from pictures")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	gathered := []analog{}

	for rows.Next() {
		var a analog
		err := rows.Scan(&a.id, &a.url, &a.title, &a.permalink, &a.score, &a.nsfw, &a.time)
		if err != nil {
			log.Fatal(err)
		}
		gathered = append(gathered, a)
	}

	fmt.Println(gathered)

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

}

func parent_dir() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	parent := filepath.Dir(wd)
	return parent
}

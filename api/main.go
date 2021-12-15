package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

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

	var (
		id  int
		url string
	)

	rows, err := db.Query("SELECT id, url from pictures WHERE id = ?", 1)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&id, &url)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(id, url)
	}

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

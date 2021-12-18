package main

import (
	"go-reddit/models"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db_path := parent_dir() + "/test.db"
	err := models.InitDB(db_path)
	if err != nil {
		log.Fatal(err)
	}
	models.AllPosts()
}

func parent_dir() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	parent := filepath.Dir(wd)
	return parent
}

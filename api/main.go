package main

import (
	"go-reddit/models"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {

	db_path := parent_dir() + "/test.db"
	err := models.InitDB(db_path)
	if err != nil {
		log.Fatal(err)
	}
	// models.AllPosts()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", helloWorld)
	r.Get("/latest", getLatest)
	http.ListenAndServe(":3000", r)
}

func getLatest(w http.ResponseWriter, r *http.Request) {
	latest, err := models.LatestPost()
	if err != nil {
		log.Fatal(err)
	}
	w.Write([]byte(latest.Url))
}

func helloWorld(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World"))
}

func parent_dir() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	parent := filepath.Dir(wd)
	return parent
}

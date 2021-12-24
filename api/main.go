package main

import (
	"fmt"
	"go-reddit/models"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	models.LoadEnv()

	db_path := parent_dir() + "/test.db"
	err := models.InitDB(db_path)
	if err != nil {
		log.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", helloWorld)
	r.Get("/latest", getLatest)
	r.Get("/random", getRandom)

	fmt.Println("listening...")
	http.ListenAndServe(getPort(), r)
}

func getLatest(w http.ResponseWriter, r *http.Request) {
	latest, err := models.LatestPost()
	if err != nil {
		log.Fatal(err)
	}
	w.Write([]byte(latest.Url))
}

func getRandom(w http.ResponseWriter, r *http.Request) {
	random, err := models.RandomPost()
	if err != nil {
		log.Fatal(err)
	}
	w.Write([]byte(random.Url))
}

func helloWorld(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World!"))
}

func parent_dir() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	parent := filepath.Dir(wd)
	return parent
}

func getPort() string {
	var port = os.Getenv("PORT")
	if port == "" {
		port = "3000"
		fmt.Println("No PORT env variable found, defaulting to: " + port)
	}
	return ":" + port
}

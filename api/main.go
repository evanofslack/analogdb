package main

import (
	"fmt"
	"go-reddit/models"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	err := models.InitDB(true)
	if err != nil {
		log.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", helloWorld)

	r.Route("/latest", func(r chi.Router) {
		r.Get("/", getLatestPost)
		r.Get("/{num}", getLatestPost)
	})
	r.Route("/top", func(r chi.Router) {
		r.Get("/", getTopPost)
		r.Get("/{num}", getTopPost)
	})
	r.Route("/random", func(r chi.Router) {
		r.Get("/", getRandomPost)
		r.Get("/{num}", getRandomPost)
	})

	fmt.Println("listening...")
	http.ListenAndServe(getPort(), r)
}

func getPort() string {
	var port = os.Getenv("PORT")
	if port == "" {
		port = "3000"
		fmt.Println("No PORT env variable found, defaulting to: " + port)
	}
	return ":" + port
}

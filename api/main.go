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
	err := models.InitDB()
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

func getPort() string {
	var port = os.Getenv("PORT")
	if port == "" {
		port = "3000"
		fmt.Println("No PORT env variable found, defaulting to: " + port)
	}
	return ":" + port
}

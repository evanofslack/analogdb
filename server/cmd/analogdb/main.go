package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	mw "github.com/evanofslack/analogdb/middleware"
	"github.com/evanofslack/analogdb/models"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	err = models.InitDB(false)
	if err != nil {
		log.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "DELETE"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           500,
	}))

	r.Handle("/*", http.FileServer(http.Dir("./static")))

	r.Group(func(r chi.Router) {
		r.Use(mw.Pagination)
		r.Get("/latest", listLatest)
		r.Get("/top", listTop)
		r.Get("/random", listRandom)
		r.Get("/nsfw", listNsfw)
		r.Get("/bw", listBw)
		r.Get("/sprocket", listSprocket)
	})

	r.Route("/posts/{id}", func(r chi.Router) {
		r.Get("/", findPost)
		r.With(mw.BasicAuth).Delete("/", deletePost)
	})

	fmt.Println("listening...")
	http.ListenAndServe(getPort(), r)
}

func getPort() string {
	var port = os.Getenv("PORT")
	if port == "" {
		port = "5000"
		fmt.Println("No PORT env variable found, defaulting to: " + port)
	}
	return ":" + port
}

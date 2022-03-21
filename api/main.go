package main

import (
	"fmt"
	mw "go-reddit/middleware"
	"go-reddit/models"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	err := models.InitDB(true
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
		r.Get("/randon", listRandom)
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

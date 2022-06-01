package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	router *chi.Mux
	port   string

	PostService analogdb.PostService
}

func New() *Server {
	s := &Server{
		router: chi.NewRouter(),
		port:   getPort(),
	}
	return s
}

func getPort() string {
	var port = os.Getenv("PORT")
	if port == "" {
		port = "8080"
		fmt.Println("No PORT env variable found, defaulting to: " + port)
	}
	return ":" + port
}

func (s *Server) Run() {
	fmt.Println("starting server...")
	http.ListenAndServe(s.port, s.router)
}

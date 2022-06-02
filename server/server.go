package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
)

const shutdownTimeout = 2 * time.Second

type Server struct {
	server *http.Server
	router *chi.Mux
	port   string

	PostService analogdb.PostService
}

func New() *Server {
	s := &Server{
		server: &http.Server{},
		router: chi.NewRouter(),
		port:   getPort(),
	}

	s.server.Handler = s.router
	// s.server.Handler = http.HandlerFunc(s.router.ServeHTTP)
	s.mountMiddleware()
	s.mountPostHandlers()
	s.mountStatic()
	s.mountStatus()
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
	// s.server.ListenAndServe()
	http.ListenAndServe(s.port, s.router)
}

func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return s.server.Shutdown(ctx)
}

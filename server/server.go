package server

import (
	"context"
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

	PostService analogdb.PostService
}

func New() *Server {
	s := &Server{
		server: &http.Server{},
		router: chi.NewRouter(),
	}

	s.server.Handler = s.router
	s.server.Addr = getPort()

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
	}
	return ":" + port
}

func (s *Server) Run() error {
	go s.server.ListenAndServe()
	return nil
}

func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return s.server.Shutdown(ctx)
}

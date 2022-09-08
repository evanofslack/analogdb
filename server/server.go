package server

import (
	"context"
	"net/http"
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

func New(port string) *Server {
	s := &Server{
		server: &http.Server{},
		router: chi.NewRouter(),
	}

	s.server.Handler = s.router
	s.server.Addr = ":" + port

	s.mountMiddleware()
	s.mountPostHandlers()
	s.mountStatic()
	s.mountStatus()
	s.mountStatsHandlers()
	return s
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

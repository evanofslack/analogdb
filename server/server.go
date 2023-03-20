package server

import (
	"context"
	"net/http"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
)

const shutdownTimeout = 5 * time.Second

type Server struct {
	server  *http.Server
	router  *chi.Mux
	healthy bool

	PostService   analogdb.PostService
	ReadyService  analogdb.ReadyService
	AuthorService analogdb.AuthorService
}

func New(port string) *Server {
	s := &Server{
		server: &http.Server{},
		router: chi.NewRouter(),
	}

	s.server.Handler = s.router
	s.server.Addr = ":" + port
	s.healthy = true

	s.mountMiddleware()
	s.mountPostHandlers()
	s.mountAuthorHandlers()
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
	s.healthy = false
	return s.server.Shutdown(ctx)
}

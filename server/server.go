package server

import (
	"context"
	"encoding/json"
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

	PostService       analogdb.PostService
	ReadyService      analogdb.ReadyService
	AuthorService     analogdb.AuthorService
	ScrapeService     analogdb.ScrapeService
	SimilarityService analogdb.SimilarityService
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
	s.mountScrapeHandlers()
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

func encodeResponse(w http.ResponseWriter, r *http.Request, status int, v any) error {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return err
	}
	return nil
}

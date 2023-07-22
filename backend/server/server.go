package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/config"
	"github.com/evanofslack/analogdb/logger"
	"github.com/evanofslack/analogdb/metrics"
	"github.com/go-chi/chi/v5"
)

const shutdownTimeout = 5 * time.Second

type Server struct {
	server    *http.Server
	router    *chi.Mux
	healthy   bool
	logger    *logger.Logger
	metrics   *metrics.Metrics
	stats     *httpStats
	basicAuth *config.Auth

	PostService       analogdb.PostService
	ReadyService      analogdb.ReadyService
	AuthorService     analogdb.AuthorService
	ScrapeService     analogdb.ScrapeService
	SimilarityService analogdb.SimilarityService
}

func New(port string, logger *logger.Logger, metrics *metrics.Metrics, basicAuth *config.Auth) *Server {
	s := &Server{
		server:    &http.Server{},
		router:    chi.NewRouter(),
		logger:    logger,
		metrics:   metrics,
		basicAuth: basicAuth,
	}

	s.server.Handler = s.router
	s.server.Addr = ":" + port

	s.stats = newHttpStats()
	s.stats.register(s.metrics.Registry)

	s.mountMiddleware()
	s.mountPostHandlers()
	s.mountAuthorHandlers()
	s.mountSimilarityHandlers()
	s.mountScrapeHandlers()
	s.mountStaticHandlers()
	s.mountStatusHandlers()
	s.mountStatsHandlers()
	s.mountMetricsHandlers()

	s.healthy = true
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

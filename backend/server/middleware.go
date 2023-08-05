package server

import (
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/riandyrn/otelchi"
)

const (
	rateLimit       = 60
	rateLimitPeriod = time.Minute * 1
)

func (s *Server) mountMiddleware() {

	// add recoverer first
	s.router.Use(middleware.Recoverer)

	// collect prom metrics
	s.router.Use(s.collectStats)

	// is tracing enabled?
	// attach before logger so span id is logged
	if s.config.Tracing.Enabled {
		s.router.Use(otelchi.Middleware("http", otelchi.WithChiRoutes(s.router)))
		s.logger.Info().Msg("Added tracing middleware")
	}

	// log all requests
	s.router.Use(s.logRequests)

	// apply rate limit
	s.addRatelimiter()

	corsHandler := cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*", "http://localhost"},
		AllowedMethods:   []string{"GET", "DELETE", "PUT", "POST", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           500,
	})

	// CORS
	s.router.Use(corsHandler)
}

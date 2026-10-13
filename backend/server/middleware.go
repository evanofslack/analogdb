package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/traceid"
	"github.com/riandyrn/otelchi"
)

const (
	rateLimit       = 60
	rateLimitPeriod = time.Minute * 1
)

func (s *Server) mountMiddleware() {
	// add recoverer first
	s.router.Use(middleware.Recoverer)

	// resolve client ip from trusted proxies
	s.router.Use(s.clientIP)

	// resolve basic auth to a role
	s.router.Use(s.principal)

	// collect prom metrics
	s.router.Use(s.collectStats)

	// is tracing enabled?
	// attach before logger so span id is logged
	if s.config.Tracing.Enabled {
		s.router.Use(otelchi.Middleware("http", otelchi.WithChiRoutes(s.router)))
		s.logger.Info("Added tracing middleware")
	}

	// extract traceid
	s.router.Use(traceid.Middleware)

	// log all requests
	s.router.Use(s.logRequests)

	// apply rate limit
	s.addRatelimiter()

	corsHandler := cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "HEAD", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           3600,
	})

	// CORS
	s.router.Use(corsHandler)

	// gzip JSON last, so stats and logs count the bytes sent
	s.router.Use(s.compress)
}

// compress gzips JSON for every caller but the web, whose calls stay on the
// internal network where compressing only costs CPU
func (s *Server) compress(next http.Handler) http.Handler {
	compressed := middleware.Compress(5, "application/json")(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p := principalFrom(r); p != nil && p.role == roleWeb {
			next.ServeHTTP(w, r)
			return
		}
		compressed.ServeHTTP(w, r)
	})
}

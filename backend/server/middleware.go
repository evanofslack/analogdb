package server

import (
	"net/http"
	"time"

	"github.com/evanofslack/analogdb/logger"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
)

const (
	rateLimit       = 60
	rateLimitPeriod = time.Minute * 1
)

func (s *Server) mountMiddleware() {

	s.router.Use(middleware.Recoverer)
	s.router.Use(logger.Middleware(s.logger))
	s.router.Use(s.collectStats)

	// is rate limiting enabled?
	if s.config.App.RateLimitEnabled {

		// rate limit by IP with json response
		rateLimiter := httprate.Limit(rateLimit, rateLimitPeriod,
			httprate.WithKeyFuncs(httprate.KeyByIP),
			httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error": "Too many requests"}`))
			}))

		// bypass rate limit if authenticated
		s.router.Use(middleware.Maybe(rateLimiter, s.applyRateLimit))
		s.logger.Info().Int("limit", rateLimit).Str("period", rateLimitPeriod.String()).Msg("Added rate limiting middleware")
	}

	corsHandler := cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*", "http://localhost"},
		AllowedMethods:   []string{"GET", "DELETE", "PUT", "POST", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           500,
	})
	s.router.Use(corsHandler)
}

// apply rate limit only if user is not authenticated
func (s *Server) applyRateLimit(r *http.Request) bool {

	rl_username := s.config.Auth.RateLimitUsername
	rl_password := s.config.Auth.RateLimitPassword

	authenticated := s.passBasicAuth(rl_username, rl_password, r)
	if authenticated {
		s.logger.Debug().Bool("authenticated", authenticated).Msg("Bypassing rate limit")
		return false
	}

	s.logger.Debug().Bool("authenticated", authenticated).Msg("Applying rate limit")
	return true
}

package server

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

func (server *Server) addRatelimiter() {

	if !server.config.App.RateLimitEnabled {
		return
	}

	// rate limit by IP with json response
	rateLimiter := httprate.Limit(rateLimit, rateLimitPeriod,
		httprate.WithKeyFuncs(httprate.KeyByIP),
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "Too many requests"}`))
		}))

	server.router.Use(middleware.Maybe(rateLimiter, server.applyRateLimit))
	server.logger.Info().Msg("Added rate limiting middleware")
}

// apply rate limit only if user is not authenticated
func (server *Server) applyRateLimit(r *http.Request) bool {

	rl_username := server.config.Auth.RateLimitUsername
	rl_password := server.config.Auth.RateLimitPassword

	authenticated := server.passBasicAuth(rl_username, rl_password, r)
	if authenticated {
		return false
	}
	return true

}

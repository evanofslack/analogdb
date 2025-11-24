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
	server.logger.Info("Added rate limiting middleware")
}

// apply rate limit if user is not authenticated
func (server *Server) applyRateLimit(r *http.Request) bool {
	rl_username := server.config.Auth.RateLimitUsername
	rl_password := server.config.Auth.RateLimitPassword
	is_rl := server.passBasicAuth(rl_username, rl_password, r)

	auth_username := server.config.Auth.Username
	auth_password := server.config.Auth.Password
	is_auth := server.passBasicAuth(auth_username, auth_password, r)

	// If either user/pw combo is provided, do not ratelimit
	if is_rl || is_auth {
		return false
	}
	return true
}

package server

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"
)

const webRateLimitKey = "role:web"

func (server *Server) addRatelimiter() {
	if !server.config.App.RateLimitEnabled {
		return
	}

	limited := func(label string) httprate.Option {
		return httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			server.stats.rateLimited.WithLabelValues(label).Inc()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error": "Too many requests"}`))
		})
	}

	anonymous := httprate.LimitBy(rateLimit, rateLimitPeriod, keyByClientIP, limited("anonymous"))

	// web shares one bucket across all its visitors, a circuit breaker for the whole site
	var web func(http.Handler) http.Handler
	if perMinute := server.config.App.RateLimitWebPerMinute; perMinute > 0 {
		web = httprate.LimitBy(perMinute, time.Minute, keyWeb, limited(string(roleWeb)))
		server.logger.Info("Added web rate limit", "per_minute", perMinute)
	} else {
		server.logger.Info("Web rate limit off")
	}

	server.router.Use(func(next http.Handler) http.Handler {
		anonymousNext := anonymous(next)
		webNext := next
		if web != nil {
			webNext = web(next)
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := principalFrom(r)
			switch {
			case p == nil:
				anonymousNext.ServeHTTP(w, r)
			case p.role == roleWeb:
				webNext.ServeHTTP(w, r)
			default:
				next.ServeHTTP(w, r)
			}
		})
	})
	server.logger.Info("Added rate limiting middleware")
}

func keyByClientIP(r *http.Request) (string, error) {
	return getRealIP(r), nil
}

func keyWeb(r *http.Request) (string, error) {
	return webRateLimitKey, nil
}

package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

var sunsetDate = time.Date(2027, time.March, 31, 0, 0, 0, 0, time.UTC).Format(http.TimeFormat)

const (
	webUserAgentPrefix     = "analogdb-web/"
	scraperUserAgentPrefix = "analogdb-scraper/"
)

func (s *Server) deprecationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Deprecation", "true")
		w.Header().Set("Sunset", sunsetDate)
		w.Header().Set("Link", fmt.Sprintf("</v1%s>; rel=\"successor-version\"", r.URL.Path))
		next.ServeHTTP(w, r)

		route := "unknown"
		if rctx := chi.RouteContext(r.Context()); rctx != nil {
			route = rctx.RoutePattern()
		}
		s.stats.legacyRequests.WithLabelValues(route, legacyClient(r.UserAgent())).Inc()
	})
}

func legacyClient(userAgent string) string {
	switch {
	case strings.HasPrefix(userAgent, webUserAgentPrefix):
		return "web"
	case strings.HasPrefix(userAgent, scraperUserAgentPrefix):
		return "scraper"
	default:
		return "other"
	}
}

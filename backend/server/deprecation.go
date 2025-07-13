package server

import (
	"fmt"
	"net/http"
)

const sunsetDate = "2025-12-31"

func (s *Server) deprecationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Deprecation", "true")
		w.Header().Set("Sunset", sunsetDate)
		w.Header().Set("Link", fmt.Sprintf("</v1%s>; rel=\"successor-version\"", r.URL.Path))
		next.ServeHTTP(w, r)
	})
}

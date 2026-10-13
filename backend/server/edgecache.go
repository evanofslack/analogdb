package server

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/evanofslack/analogdb"
)

const (
	catalogEdgeTTL = time.Hour
	listEdgeTTL    = 5 * time.Minute
)

// cachePolicy returns how long the edge may cache a response, or false to not
// cache it
type cachePolicy func(r *http.Request) (time.Duration, bool)

func fixedTTL(ttl time.Duration) cachePolicy {
	return func(*http.Request) (time.Duration, bool) { return ttl, true }
}

// postsTTL skips random sort without a valid seed or cursor, since the server
// picks the seed then and a cached copy would freeze one order
func postsTTL(r *http.Request) (time.Duration, bool) {
	values := r.URL.Query()
	if values.Get("sort") != analogdb.PostSortRandom.String() || values.Get("cursor") != "" {
		return listEdgeTTL, true
	}
	seed, err := strconv.Atoi(values.Get("seed"))
	if err != nil || !validSeed(seed) {
		return 0, false
	}
	return listEdgeTTL, true
}

// edgeCache marks successful anonymous GETs as cacheable by Cloudflare for the
// policy's ttl. Everything else gets no Cache-Control and bypasses the edge
func (s *Server) edgeCache(policy cachePolicy) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet && r.Method != http.MethodHead || principalFrom(r) != nil {
				next.ServeHTTP(w, r)
				return
			}
			ttl, ok := policy(r)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(&edgeCacheWriter{ResponseWriter: w, ttl: ttl}, r)
		})
	}
}

type edgeCacheWriter struct {
	http.ResponseWriter
	ttl         time.Duration
	wroteHeader bool
}

func (w *edgeCacheWriter) WriteHeader(status int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		if status == http.StatusOK {
			secs := int(w.ttl.Seconds())
			w.Header().Set("Cache-Control", fmt.Sprintf("public, s-maxage=%d, stale-while-revalidate=%d", secs, secs))
		}
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *edgeCacheWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

func (w *edgeCacheWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *edgeCacheWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

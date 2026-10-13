package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/config"
)

const catalogCacheControl = "public, s-maxage=3600, stale-while-revalidate=3600"

func TestEdgeCacheRoutes(t *testing.T) {
	s := newRolesServer(t, &config.Config{})
	s.CameraService = &mockCameraService{cameras: []*analogdb.Camera{{Id: 1, Make: "nikon", Model: "f3"}}}
	s.PostService = &mockFindPostService{}

	tests := []struct {
		name   string
		method string
		path   string
		creds  *pair
		status int
		want   string
	}{
		{"anonymous catalog", http.MethodGet, "/v1/cameras", nil, http.StatusOK, catalogCacheControl},
		{"web catalog", http.MethodGet, "/v1/cameras", &webPair, http.StatusOK, ""},
		{"scraper catalog", http.MethodGet, "/v1/cameras", &scraperPair, http.StatusOK, ""},
		{"bad request", http.MethodGet, "/v1/cameras?sort=bogus", nil, http.StatusBadRequest, ""},
		{"posts default sort", http.MethodGet, "/v1/posts", nil, http.StatusOK, "public, s-maxage=300, stale-while-revalidate=300"},
		{"posts random without seed", http.MethodGet, "/v1/posts?sort=random", nil, http.StatusOK, ""},
		{"posts random invalid seed", http.MethodGet, "/v1/posts?sort=random&seed=0", nil, http.StatusOK, ""},
		{"posts random with seed", http.MethodGet, "/v1/posts?sort=random&seed=7", nil, http.StatusOK, "public, s-maxage=300, stale-while-revalidate=300"},
		{"admin route", http.MethodGet, "/v1/admin/overview", nil, http.StatusUnauthorized, "no-store"},
		{"scrape route", http.MethodGet, "/v1/scrape/vectors/missing", nil, http.StatusUnauthorized, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := roleRequest(s, tt.method, tt.path, tt.creds, "")
			if w.Code != tt.status {
				t.Fatalf("want status %d, got %d: %s", tt.status, w.Code, w.Body.String())
			}
			if got := w.Header().Get("Cache-Control"); got != tt.want {
				t.Errorf("want Cache-Control %q, got %q", tt.want, got)
			}
		})
	}
}

func TestEdgeCacheWriter(t *testing.T) {
	s := &Server{}
	tests := []struct {
		name   string
		method string
		status int
		write  bool
		want   string
	}{
		{"ok", http.MethodGet, http.StatusOK, false, "public, s-maxage=60, stale-while-revalidate=60"},
		{"implicit ok", http.MethodGet, 0, true, "public, s-maxage=60, stale-while-revalidate=60"},
		{"rate limited", http.MethodGet, http.StatusTooManyRequests, false, ""},
		{"server error", http.MethodGet, http.StatusInternalServerError, false, ""},
		{"head", http.MethodHead, http.StatusOK, false, "public, s-maxage=60, stale-while-revalidate=60"},
		{"post", http.MethodPost, http.StatusOK, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := s.edgeCache(fixedTTL(time.Minute))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.status != 0 {
					w.WriteHeader(tt.status)
				}
				if tt.write {
					_, _ = w.Write([]byte("{}"))
				}
			}))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(tt.method, "/", nil))
			if got := w.Header().Get("Cache-Control"); got != tt.want {
				t.Errorf("want Cache-Control %q, got %q", tt.want, got)
			}
		})
	}
}

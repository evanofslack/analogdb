package server

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/config"
)

func compressRequest(s *Server, path string, creds *pair, encoding string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	if creds != nil {
		r.SetBasicAuth(creds.username, creds.password)
	}
	if encoding != "" {
		r.Header.Set("Accept-Encoding", encoding)
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)
	return w
}

func gunzip(t *testing.T, w *httptest.ResponseRecorder) []byte {
	t.Helper()
	zr, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatalf("body is not gzip: %v", err)
	}
	body, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	if err := zr.Close(); err != nil {
		t.Fatal(err)
	}
	return body
}

func TestCompress(t *testing.T) {
	s := newRolesServer(t, &config.Config{})
	s.CameraService = &mockCameraService{cameras: []*analogdb.Camera{{Id: 1, Make: "nikon", Model: "f3"}}}

	t.Run("anonymous gzip", func(t *testing.T) {
		w := compressRequest(s, "/v1/cameras", nil, "gzip")
		if w.Code != http.StatusOK {
			t.Fatalf("want 200, got %d", w.Code)
		}
		if got := w.Header().Get("Content-Encoding"); got != "gzip" {
			t.Fatalf("want gzip, got %q", got)
		}
		if !slices.Contains(w.Header().Values("Vary"), "Accept-Encoding") {
			t.Errorf("want Vary Accept-Encoding, got %q", w.Header().Values("Vary"))
		}
		var resp CamerasResponse
		if err := json.Unmarshal(gunzip(t, w), &resp); err != nil {
			t.Fatalf("gunzipped body is not json: %v", err)
		}
		if len(resp.Cameras) != 1 {
			t.Errorf("want 1 camera, got %d", len(resp.Cameras))
		}
	})

	t.Run("no accept encoding", func(t *testing.T) {
		w := compressRequest(s, "/v1/cameras", nil, "")
		if got := w.Header().Get("Content-Encoding"); got != "" {
			t.Fatalf("want no encoding, got %q", got)
		}
		if !json.Valid(w.Body.Bytes()) {
			t.Errorf("want plain json, got %q", w.Body.String())
		}
	})

	t.Run("web role stays plain", func(t *testing.T) {
		w := compressRequest(s, "/v1/cameras", &webPair, "gzip")
		if w.Code != http.StatusOK {
			t.Fatalf("want 200, got %d", w.Code)
		}
		if got := w.Header().Get("Content-Encoding"); got != "" {
			t.Fatalf("want no encoding for web, got %q", got)
		}
		if !json.Valid(w.Body.Bytes()) {
			t.Errorf("want plain json, got %q", w.Body.String())
		}
	})

	t.Run("scraper gzip", func(t *testing.T) {
		w := compressRequest(s, "/v1/cameras", &scraperPair, "gzip")
		if got := w.Header().Get("Content-Encoding"); got != "gzip" {
			t.Fatalf("want gzip for scraper, got %q", got)
		}
	})

	t.Run("error body gzip", func(t *testing.T) {
		w := compressRequest(s, "/v1/cameras?sort=bogus", nil, "gzip")
		if w.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d", w.Code)
		}
		if got := w.Header().Get("Content-Encoding"); got != "gzip" {
			t.Fatalf("want gzip, got %q", got)
		}
		var resp ErrorResponse
		if err := json.Unmarshal(gunzip(t, w), &resp); err != nil {
			t.Fatalf("gunzipped error is not json: %v", err)
		}
	})
}

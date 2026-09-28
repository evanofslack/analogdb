package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReady(t *testing.T) {
	s := mustOpen(t)
	defer mustClose(t, s)
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)

	if want, got := http.StatusOK, w.Code; got != want {
		t.Errorf("want status %d, got %d", want, got)
	}
}

func TestHealthy(t *testing.T) {
	s := mustOpen(t)
	defer mustClose(t, s)
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)

	if want, got := http.StatusOK, w.Code; got != want {
		t.Errorf("want status %d, got %d", want, got)
	}
}

func TestUnhealthy(t *testing.T) {
	s := mustOpen(t)
	defer mustClose(t, s)
	s.healthy.Store(false)
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)

	if want, got := http.StatusServiceUnavailable, w.Code; got != want {
		t.Errorf("want status %d, got %d", want, got)
	}

	var resp ErrorResponse
	dec := json.NewDecoder(w.Body)
	if err := dec.Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if dec.More() {
		t.Error("want single response body")
	}
}

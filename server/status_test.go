package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReady(t *testing.T) {
	s, db := mustOpen(t)
	defer mustClose(t, s, db)
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)

	if want, got := http.StatusOK, w.Code; got != want {
		t.Errorf("want status %d, got %d", want, got)
	}
}

func TestHealthy(t *testing.T) {
	s, db := mustOpen(t)
	defer mustClose(t, s, db)
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)

	if want, got := http.StatusOK, w.Code; got != want {
		t.Errorf("want status %d, got %d", want, got)
	}
}

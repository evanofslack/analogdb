package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestKeywordSummaryInvalidPageSize(t *testing.T) {
	s := mustOpen(t)
	defer mustClose(t, s)
	r := httptest.NewRequest(http.MethodGet, "/keywords/summary?page_size=abc", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)

	if want, got := http.StatusBadRequest, w.Code; got != want {
		t.Errorf("want status %d, got %d", want, got)
	}
}

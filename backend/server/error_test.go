package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/logger"
)

func TestWriteErrorLevels(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantLevel  string
	}{
		{name: "not found", err: &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "post not found"}, wantStatus: http.StatusNotFound, wantLevel: "INFO"},
		{name: "bad request", err: badRequest("bad"), wantStatus: http.StatusBadRequest, wantLevel: "INFO"},
		{name: "unavailable", err: &analogdb.Error{Code: analogdb.ERRUNAVAILABLE, Message: "down"}, wantStatus: http.StatusServiceUnavailable, wantLevel: "ERROR"},
		{name: "internal", err: errors.New("boom"), wantStatus: http.StatusInternalServerError, wantLevel: "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
			s := &Server{logger: &logger.Logger{Logger: slog.New(handler)}}

			w := httptest.NewRecorder()
			s.writeError(w, httptest.NewRequest(http.MethodGet, "/v1/post/1", nil), tt.err)

			if w.Code != tt.wantStatus {
				t.Errorf("want status %d, got %d", tt.wantStatus, w.Code)
			}
			var entry map[string]any
			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatal(err)
			}
			if entry["level"] != tt.wantLevel {
				t.Errorf("want level %s, got %v", tt.wantLevel, entry["level"])
			}
		})
	}
}

func TestHandleRequestLoggedAtDebug(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	s := &Server{logger: &logger.Logger{Logger: slog.New(handler)}, EventService: &recordingEvents{}}

	h := s.logRequests(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/v1/posts", nil))

	if bytes.Contains(buf.Bytes(), []byte("Handle request")) {
		t.Errorf("want request log hidden at info level, got %s", buf.String())
	}
}

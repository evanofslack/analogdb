package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evanofslack/analogdb/events"
	v1 "github.com/evanofslack/analogdb/internal/gen/proto/analytics/v1"
	"github.com/evanofslack/analogdb/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/kafka-go"
)

type blockingWriter struct{}

func (blockingWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	select {
	case <-time.After(5 * time.Second):
	case <-ctx.Done():
	}
	return ctx.Err()
}

func (blockingWriter) Close() error { return nil }

func TestLogRequestsDoesNotWaitOnEvents(t *testing.T) {
	l, err := logger.New("error", "debug", "analogdb-test")
	if err != nil {
		t.Fatal(err)
	}
	es, err := events.NewWithWriter(l, prometheus.NewRegistry(), blockingWriter{}, events.Options{QueueSize: 1, BatchSize: 1})
	if err != nil {
		t.Fatal(err)
	}

	s := &Server{logger: l, EventService: es}
	handler := s.logRequests(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 5; i++ {
		start := time.Now()
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/posts", nil))
		if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
			t.Fatalf("request %d took %s, want under 50ms", i, elapsed)
		}
	}
}

type recordingEvents struct {
	events []*v1.Event
}

func (r *recordingEvents) Write(ctx context.Context, event *v1.Event) error {
	r.events = append(r.events, event)
	return nil
}

func (r *recordingEvents) Close() error { return nil }

func TestLogRequestsAuthorized(t *testing.T) {
	s := mustOpen(t)
	defer mustClose(t, s)
	s.config.Auth.Username = "admin"
	s.config.Auth.Password = "secret"
	rec := &recordingEvents{}
	s.EventService = rec

	tests := []struct {
		name       string
		path       string
		password   string
		wantStatus int
		wantAuth   bool
	}{
		{name: "authed route good creds", path: "/v1/scrape/keywords/updated", password: "secret", wantStatus: http.StatusOK, wantAuth: true},
		{name: "authed route bad creds", path: "/v1/scrape/keywords/updated", password: "wrong", wantStatus: http.StatusUnauthorized, wantAuth: false},
		{name: "public route", path: "/ping", password: "secret", wantStatus: http.StatusOK, wantAuth: false},
	}
	s.ScrapeService = mockScrape{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec.events = nil
			r := httptest.NewRequest(http.MethodGet, tt.path, nil)
			r.SetBasicAuth("admin", tt.password)
			w := httptest.NewRecorder()
			s.router.ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Fatalf("want status %d, got %d", tt.wantStatus, w.Code)
			}
			if len(rec.events) != 1 {
				t.Fatalf("want 1 event, got %d", len(rec.events))
			}
			if got := rec.events[0].Authorized; got != tt.wantAuth {
				t.Errorf("want authorized %v, got %v", tt.wantAuth, got)
			}
		})
	}
}

type mockScrape struct{}

func (mockScrape) KeywordUpdatedPostIDs(ctx context.Context) ([]int, error) {
	return []int{1}, nil
}

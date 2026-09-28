package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evanofslack/analogdb/events"
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

package events

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	v1 "github.com/evanofslack/analogdb/internal/gen/proto/analytics/v1"
	"github.com/evanofslack/analogdb/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/segmentio/kafka-go"
)

type fakeWriter struct {
	mu      sync.Mutex
	delay   time.Duration
	written []kafka.Message
	closed  bool
}

func (w *fakeWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	select {
	case <-time.After(w.delay):
	case <-ctx.Done():
		return ctx.Err()
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.written = append(w.written, msgs...)
	return nil
}

func (w *fakeWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	return nil
}

func (w *fakeWriter) count() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.written)
}

func newTestStream(t *testing.T, writer MessageWriter, opts Options) *EventStream {
	t.Helper()
	l, err := logger.New("error", "debug", "analogdb-test")
	if err != nil {
		t.Fatal(err)
	}
	es, err := NewWithWriter(l, prometheus.NewRegistry(), writer, opts)
	if err != nil {
		t.Fatal(err)
	}
	return es
}

func TestWriteDoesNotBlock(t *testing.T) {
	writer := &fakeWriter{delay: 5 * time.Second}
	es := newTestStream(t, writer, Options{QueueSize: 2, BatchSize: 1, BatchTimeout: 10 * time.Millisecond})
	defer es.cancel()

	if err := es.Write(context.Background(), &v1.Event{Path: "/first"}); err != nil {
		t.Fatal(err)
	}
	// let the worker pick up the first event and block in the writer
	time.Sleep(50 * time.Millisecond)

	var fullErr error
	for i := 0; i < 5; i++ {
		start := time.Now()
		err := es.Write(context.Background(), &v1.Event{Path: "/next"})
		if elapsed := time.Since(start); elapsed > time.Millisecond {
			t.Fatalf("write took %s, want under 1ms", elapsed)
		}
		if err != nil {
			fullErr = err
		}
	}
	if !errors.Is(fullErr, ErrQueueFull) {
		t.Fatalf("want ErrQueueFull, got %v", fullErr)
	}
	if got := testutil.ToFloat64(es.stats.dropped); got != 3 {
		t.Errorf("want 3 dropped, got %v", got)
	}
	if got := testutil.ToFloat64(es.stats.enqueued); got != 3 {
		t.Errorf("want 3 enqueued, got %v", got)
	}
	if got := testutil.ToFloat64(es.stats.queueDepth); got != 2 {
		t.Errorf("want queue depth 2, got %v", got)
	}
}

func TestCloseDrainsQueue(t *testing.T) {
	writer := &fakeWriter{delay: 10 * time.Millisecond}
	es := newTestStream(t, writer, Options{QueueSize: 100, BatchSize: 10, BatchTimeout: time.Hour})

	for i := 0; i < 25; i++ {
		if err := es.Write(context.Background(), &v1.Event{Path: "/posts"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := es.Close(); err != nil {
		t.Fatal(err)
	}
	if got := writer.count(); got != 25 {
		t.Errorf("want 25 written, got %d", got)
	}
	if !writer.closed {
		t.Error("want writer closed")
	}
	if err := es.Write(context.Background(), &v1.Event{}); !errors.Is(err, ErrClosed) {
		t.Errorf("want ErrClosed after close, got %v", err)
	}
	if err := es.Close(); err != nil {
		t.Errorf("second close: %v", err)
	}
}

func TestBatchTimeoutFlushes(t *testing.T) {
	writer := &fakeWriter{}
	es := newTestStream(t, writer, Options{QueueSize: 100, BatchSize: 100, BatchTimeout: 20 * time.Millisecond})
	defer func() { _ = es.Close() }()

	for i := 0; i < 3; i++ {
		if err := es.Write(context.Background(), &v1.Event{}); err != nil {
			t.Fatal(err)
		}
	}
	deadline := time.Now().Add(time.Second)
	for writer.count() < 3 {
		if time.Now().After(deadline) {
			t.Fatalf("want 3 written after batch timeout, got %d", writer.count())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestCloseDeadline(t *testing.T) {
	writer := &fakeWriter{delay: time.Minute}
	es := newTestStream(t, writer, Options{QueueSize: 10, BatchSize: 1, BatchTimeout: time.Hour})

	for i := 0; i < 5; i++ {
		if err := es.Write(context.Background(), &v1.Event{}); err != nil {
			t.Fatal(err)
		}
	}
	start := time.Now()
	if err := es.Close(); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > drainTimeout+time.Second {
		t.Errorf("close took %s, want about %s", elapsed, drainTimeout)
	}
	if got := testutil.ToFloat64(es.stats.writeErrors); got != 5 {
		t.Errorf("want 5 write errors, got %v", got)
	}
}

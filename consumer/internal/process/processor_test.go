package process

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"

	v1 "github.com/evanofslack/analogdb-consumer/internal/gen/proto/analytics/v1"
	"github.com/evanofslack/analogdb-consumer/internal/metrics"
)

type batch struct {
	events   []*v1.UiEvent
	messages []kafka.Message
}

type fakeConsumer struct {
	cancel      context.CancelFunc
	batches     []batch
	reads       int
	commits     [][]kafka.Message
	commitFails int
}

func (f *fakeConsumer) Read(ctx context.Context) ([]*v1.UiEvent, []kafka.Message, error) {
	f.reads++
	if len(f.batches) == 0 {
		f.cancel()
		return nil, nil, ctx.Err()
	}
	b := f.batches[0]
	f.batches = f.batches[1:]
	return b.events, b.messages, nil
}

func (f *fakeConsumer) Commit(ctx context.Context, msgs []kafka.Message) error {
	if f.commitFails > 0 {
		f.commitFails--
		return errors.New("commit failed")
	}
	f.commits = append(f.commits, msgs)
	return nil
}

func (f *fakeConsumer) Close() error { return nil }

type fakeDB struct {
	inserts [][]*v1.UiEvent
	fails   int
	onFail  func()
}

func (f *fakeDB) Insert(ctx context.Context, events []*v1.UiEvent) error {
	f.inserts = append(f.inserts, events)
	if f.fails > 0 {
		f.fails--
		if f.onFail != nil {
			f.onFail()
		}
		return errors.New("insert failed")
	}
	return nil
}

const (
	id1 = "6f1c3a52-6a0e-4c43-9a4b-1f0b5c1e2d01"
	id2 = "6f1c3a52-6a0e-4c43-9a4b-1f0b5c1e2d02"
)

func newBatch(events ...*v1.UiEvent) batch {
	b := batch{events: events}
	for i := range events {
		b.messages = append(b.messages, kafka.Message{Offset: int64(i)})
	}
	return b
}

func setup(t *testing.T, b batch) (*Processor[v1.UiEvent], *fakeConsumer, *fakeDB, *metrics.Metrics, context.Context) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	m, err := metrics.New(slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	consumer := &fakeConsumer{cancel: cancel, batches: []batch{b}}
	db := &fakeDB{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	p := New(logger, m, "ui", consumer, db, ValidateUiEvent)
	p.backoff = func(int) time.Duration { return 0 }
	return p, consumer, db, m, ctx
}

func run(t *testing.T, ctx context.Context, p *Processor[v1.UiEvent]) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- p.Start(ctx) }()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Start returned %v, want context canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return")
	}
}

func TestValidBatchInsertsAndCommits(t *testing.T) {
	b := newBatch(
		&v1.UiEvent{EventId: id1, EventName: "page_view"},
		&v1.UiEvent{EventId: id2, EventName: "page_view"},
	)
	p, consumer, db, _, ctx := setup(t, b)
	run(t, ctx, p)

	if len(db.inserts) != 1 || len(db.inserts[0]) != 2 {
		t.Fatalf("inserts = %v, want one insert of 2", db.inserts)
	}
	if len(consumer.commits) != 1 || len(consumer.commits[0]) != 2 {
		t.Fatalf("commits = %v, want one commit of 2", consumer.commits)
	}
}

func TestInvalidEventDropped(t *testing.T) {
	b := newBatch(
		&v1.UiEvent{EventId: id1, EventName: "page_view"},
		&v1.UiEvent{EventId: "not-a-uuid", EventName: "page_view"},
		&v1.UiEvent{EventId: id2},
	)
	p, consumer, db, m, ctx := setup(t, b)
	run(t, ctx, p)

	if len(db.inserts) != 1 || len(db.inserts[0]) != 1 || db.inserts[0][0].EventId != id1 {
		t.Fatalf("inserts = %v, want only %s", db.inserts, id1)
	}
	if len(consumer.commits) != 1 || len(consumer.commits[0]) != 3 {
		t.Fatalf("commits = %v, want one commit of 3", consumer.commits)
	}
	if got := droppedCount(t, m, "ui", "invalid"); got != 2 {
		t.Fatalf("dropped = %v, want 2", got)
	}
}

func TestAllInvalidCommitsWithoutInsert(t *testing.T) {
	b := newBatch(&v1.UiEvent{EventName: "page_view"})
	p, consumer, db, _, ctx := setup(t, b)
	run(t, ctx, p)

	if len(db.inserts) != 0 {
		t.Fatalf("inserts = %d, want 0", len(db.inserts))
	}
	if len(consumer.commits) != 1 {
		t.Fatalf("commits = %d, want 1", len(consumer.commits))
	}
}

func TestUndecodableOnlyBatchCommits(t *testing.T) {
	b := batch{messages: []kafka.Message{{Offset: 0}}}
	p, consumer, db, _, ctx := setup(t, b)
	run(t, ctx, p)

	if len(db.inserts) != 0 {
		t.Fatalf("inserts = %d, want 0", len(db.inserts))
	}
	if len(consumer.commits) != 1 {
		t.Fatalf("commits = %d, want 1", len(consumer.commits))
	}
}

func TestInsertRetriesSameBatch(t *testing.T) {
	b := newBatch(&v1.UiEvent{EventId: id1, EventName: "page_view"})
	p, consumer, db, _, ctx := setup(t, b)
	db.fails = 2
	run(t, ctx, p)

	if consumer.reads != 2 {
		t.Fatalf("reads = %d, want 2 (one batch, then the empty read that stops the test)", consumer.reads)
	}
	if len(db.inserts) != 3 {
		t.Fatalf("inserts = %d, want 3", len(db.inserts))
	}
	for _, events := range db.inserts {
		if len(events) != 1 || events[0] != b.events[0] {
			t.Fatalf("insert retried with different events: %v", events)
		}
	}
	if len(consumer.commits) != 1 {
		t.Fatalf("commits = %d, want 1", len(consumer.commits))
	}
}

func TestCommitRetriesWithoutInsert(t *testing.T) {
	b := newBatch(&v1.UiEvent{EventId: id1, EventName: "page_view"})
	p, consumer, db, _, ctx := setup(t, b)
	consumer.commitFails = 1
	run(t, ctx, p)

	if len(db.inserts) != 1 {
		t.Fatalf("inserts = %d, want 1", len(db.inserts))
	}
	if len(consumer.commits) != 1 {
		t.Fatalf("commits = %d, want 1", len(consumer.commits))
	}
}

func TestCancelMidRetryDoesNotCommit(t *testing.T) {
	b := newBatch(&v1.UiEvent{EventId: id1, EventName: "page_view"})
	p, consumer, db, _, ctx := setup(t, b)
	db.fails = 100
	calls := 0
	db.onFail = func() {
		calls++
		if calls == 2 {
			consumer.cancel()
		}
	}
	run(t, ctx, p)

	if len(db.inserts) != 2 {
		t.Fatalf("inserts = %d, want 2", len(db.inserts))
	}
	if len(consumer.commits) != 0 {
		t.Fatalf("commits = %d, want 0", len(consumer.commits))
	}
	if consumer.reads != 1 {
		t.Fatalf("reads = %d, want 1", consumer.reads)
	}
}

func TestBackoffCapped(t *testing.T) {
	if got := backoff(3); got != 3*time.Second {
		t.Fatalf("backoff(3) = %v, want 3s", got)
	}
	if got := backoff(100); got != maxRetryDelay {
		t.Fatalf("backoff(100) = %v, want %v", got, maxRetryDelay)
	}
}

func TestValidateEvent(t *testing.T) {
	if err := ValidateEvent(&v1.Event{RequestId: "a", StartTime: 1, EndTime: 2}); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}
	if err := ValidateEvent(&v1.Event{RequestId: "a", StartTime: 2, EndTime: 1}); err == nil {
		t.Fatal("end before start accepted")
	}
}

func droppedCount(t *testing.T, m *metrics.Metrics, topic, reason string) float64 {
	t.Helper()
	families, err := m.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() != "analogdb_consumer_events_dropped_total" {
			continue
		}
		for _, metric := range family.GetMetric() {
			labels := map[string]string{}
			for _, label := range metric.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}
			if labels["topic"] == topic && labels["reason"] == reason {
				return metric.GetCounter().GetValue()
			}
		}
	}
	return 0
}

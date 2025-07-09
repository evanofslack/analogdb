package metrics

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestNew(t *testing.T) {
	logger := slog.Default()
	m, err := New(logger)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if m == nil {
		t.Fatal("New() returned nil metrics")
	}
	if m.eventsRead == nil {
		t.Fatal("eventsRead metric is nil")
	}
	if m.eventsCommitted == nil {
		t.Fatal("eventsCommitted metric is nil")
	}
	if m.clickhouseInserts == nil {
		t.Fatal("clickhouseInserts metric is nil")
	}
	if m.clickhouseInsertDuration == nil {
		t.Fatal("clickhouseInsertDuration metric is nil")
	}
}

func TestIncrementEventsRead(t *testing.T) {
	logger := slog.Default()
	m, err := New(logger)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	m.IncrementEventsRead(5, "group1", "topic1", nil)
	m.IncrementEventsRead(3, "group1", "topic1", errors.New("test error"))

	metric := &dto.Metric{}
	m.eventsRead.WithLabelValues("group1", "topic1", "success").Write(metric)
	if metric.Counter.GetValue() != 5 {
		t.Errorf("Expected success count 5, got %v", metric.Counter.GetValue())
	}

	metric = &dto.Metric{}
	m.eventsRead.WithLabelValues("group1", "topic1", "fail").Write(metric)
	if metric.Counter.GetValue() != 3 {
		t.Errorf("Expected fail count 3, got %v", metric.Counter.GetValue())
	}
}

func TestIncrementEventsCommitted(t *testing.T) {
	logger := slog.Default()
	m, err := New(logger)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	m.IncrementEventsCommitted(2, "group1", "topic1", nil)
	m.IncrementEventsCommitted(1, "group1", "topic1", errors.New("test error"))

	metric := &dto.Metric{}
	m.eventsRead.WithLabelValues("group1", "topic1", "success").Write(metric)
	if metric.Counter.GetValue() != 2 {
		t.Errorf("Expected success count 2, got %v", metric.Counter.GetValue())
	}

	metric = &dto.Metric{}
	m.eventsRead.WithLabelValues("group1", "topic1", "fail").Write(metric)
	if metric.Counter.GetValue() != 1 {
		t.Errorf("Expected fail count 1, got %v", metric.Counter.GetValue())
	}
}

func TestIncrementClickHouseInserts(t *testing.T) {
	logger := slog.Default()
	m, err := New(logger)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	m.IncrementClickHouseInserts(10, "table1", nil)
	m.IncrementClickHouseInserts(5, "table1", errors.New("test error"))

	metric := &dto.Metric{}
	m.clickhouseInserts.WithLabelValues("table1", "success").Write(metric)
	if metric.Counter.GetValue() != 10 {
		t.Errorf("Expected success count 10, got %v", metric.Counter.GetValue())
	}

	metric = &dto.Metric{}
	m.clickhouseInserts.WithLabelValues("table1", "fail").Write(metric)
	if metric.Counter.GetValue() != 5 {
		t.Errorf("Expected fail count 5, got %v", metric.Counter.GetValue())
	}
}

func TestObserveClickHouseInsertDuration(t *testing.T) {
	logger := slog.Default()
	m, err := New(logger)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	duration := 250 * time.Millisecond
	m.ObserveClickHouseInsertDuration("table1", duration)

	metric := &dto.Metric{}
	histogram := m.clickhouseInsertDuration.WithLabelValues("table1").(prometheus.Histogram)
	histogram.Write(metric)
	if metric.Histogram.GetSampleCount() != 1 {
		t.Errorf("Expected sample count 1, got %v", metric.Histogram.GetSampleCount())
	}
	if metric.Histogram.GetSampleSum() != 0.25 {
		t.Errorf("Expected sample sum 0.25, got %v", metric.Histogram.GetSampleSum())
	}
}

func TestServe(t *testing.T) {
	logger := slog.Default()
	m, err := New(logger)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- m.Serve(ctx, "0")
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Serve() failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Error("Serve() did not shutdown gracefully")
	}
}

func TestRegistrationFailure(t *testing.T) {
	reg := prometheus.NewRegistry()
	prefixReg := prometheus.WrapRegistererWithPrefix("analogdb_consumer", reg)

	eventsRead := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_read_total",
			Help: "Total number of events read from Kafka",
		},
		[]string{"consumer_group", "topic", "result"},
	)

	err := prefixReg.Register(eventsRead)
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	err = prefixReg.Register(eventsRead)
	if err == nil {
		t.Error("Expected registration to fail due to duplicate metric")
	}
}

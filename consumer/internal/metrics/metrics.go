package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const shutdownTimeout = 5 * time.Second

type Metrics struct {
	logger     *slog.Logger
	registry   *prometheus.Registry
	registerer prometheus.Registerer
	server     *http.Server

	eventsRead               *prometheus.CounterVec
	eventsCommitted          *prometheus.CounterVec
	clickhouseInserts        *prometheus.CounterVec
	clickhouseInsertDuration *prometheus.HistogramVec
}

func New(logger *slog.Logger) *Metrics {
	reg := prometheus.NewRegistry()
	prefixReg := prometheus.WrapRegistererWithPrefix("analogdb_consumer", reg)
	m := &Metrics{
		logger:     logger,
		registry:   reg,
		registerer: prefixReg,
		eventsRead: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "events_read_total",
				Help: "Total number of events read from Kafka",
			},
			[]string{"consumer_group", "topic", "result"},
		),
		eventsCommitted: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "events_committed_total",
				Help: "Total number of events committed from Kafka",
			},
			[]string{"consumer_group", "topic", "result"},
		),
		clickhouseInserts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "clickhouse_inserts_total",
				Help: "Total number of successful ClickHouse inserts",
			},
			[]string{"table", "result"},
		),
		clickhouseInsertDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "clickhouse_insert_duration_seconds",
				Help:    "Time taken for ClickHouse insert operations",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"table"},
		),
	}

	m.registry.MustRegister(m.eventsRead)
	m.registry.MustRegister(m.eventsCommitted)
	m.registry.MustRegister(m.clickhouseInserts)
	m.registry.MustRegister(m.clickhouseInsertDuration)

	return m
}

func (m *Metrics) IncrementEventsRead(count int, consumerGroup, topic string, err error) {
	if err == nil {
		m.eventsRead.WithLabelValues(consumerGroup, topic, "success").Add(float64(count))
	} else {
		m.eventsRead.WithLabelValues(consumerGroup, topic, "fail").Add(float64(count))
	}
}

func (m *Metrics) IncrementEventsCommitted(count int, consumerGroup, topic string, err error) {
	if err == nil {
		m.eventsRead.WithLabelValues(consumerGroup, topic, "success").Add(float64(count))
	} else {
		m.eventsRead.WithLabelValues(consumerGroup, topic, "fail").Add(float64(count))
	}
}

func (m *Metrics) IncrementClickHouseInserts(count int, table string, err error) {
	if err == nil {
		m.clickhouseInserts.WithLabelValues(table, "success").Add(float64(count))
	} else {
		m.clickhouseInserts.WithLabelValues(table, "fail").Add(float64(count))
	}
}

func (m *Metrics) ObserveClickHouseInsertDuration(table string, duration time.Duration) {
	m.clickhouseInsertDuration.WithLabelValues(table).Observe(duration.Seconds())
}

const metricsPath = "/metrics"

func (m *Metrics) Serve(ctx context.Context, port string) error {
	mux := http.NewServeMux()
	mux.Handle(metricsPath, promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}))
	addr := ":" + port
	m.server = &http.Server{Addr: addr, Handler: mux}
	m.logger.Info("Serving prometheus metrics server", "addr", m.server.Addr, "path", metricsPath)

	errChan := make(chan error, 1)
	go func() {
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("server failed to start: %w", err)
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return m.Close(context.Background())
	}
}

func (m *Metrics) Close(ctx context.Context) error {
	m.logger.Debug("Starting prometheus metrics server close")
	defer m.logger.Info("Closed prometheus metrics server")

	ctxShutdown, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return m.server.Shutdown(ctxShutdown)
}

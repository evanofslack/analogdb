package metrics

import (
	"github.com/evanofslack/analogdb/logger"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	analogdbNamespace = "analogdb"
	httpSubsystem     = "http"
)

type stats struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	RequestSize     *prometheus.SummaryVec
	ResponseSize    *prometheus.SummaryVec
}

func NewStats() (*stats, error) {

	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: analogdbNamespace,
			Subsystem: httpSubsystem,
			Name:      "requests_total",
			Help:      "Number of HTTP requests",
		}, []string{"method", "code", "path"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: analogdbNamespace,
			Subsystem: httpSubsystem,
			Name:      "request_duration_seconds",
			Help:      "Latencies for HTTP requests",
		},
		[]string{"method", "code", "path"},
	)

	requestSize := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: analogdbNamespace,
			Subsystem: httpSubsystem,
			Name:      "request_size_bytes",
			Help:      "Size of HTTP requests",
		},
		[]string{"method", "code", "path"},
	)

	responseSize := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: analogdbNamespace,
			Subsystem: httpSubsystem,
			Name:      "response_size_bytes",
			Help:      "Size of HTTP responses",
		},
		[]string{"method", "code", "path"},
	)

	stats := &stats{
		RequestsTotal:   requestsTotal,
		RequestDuration: requestDuration,
		RequestSize:     requestSize,
		ResponseSize:    responseSize,
	}

	return stats, nil
}

func (stats *stats) register(registerer prometheus.Registerer) error {
	registerer.MustRegister(stats.RequestsTotal)
	registerer.MustRegister(stats.RequestDuration)
	registerer.MustRegister(stats.RequestSize)
	registerer.MustRegister(stats.ResponseSize)
	return nil
}

type Metrics struct {
	Registry *prometheus.Registry
	Stats    *stats
	logger   *logger.Logger
}

func New(logger *logger.Logger) (*Metrics, error) {

	registry := prometheus.NewRegistry()

	stats, err := NewStats()
	if err != nil {
		return nil, err
	}

	if err := stats.register(registry); err != nil {
		return nil, err
	}

	metrics := &Metrics{
		Registry: registry,
		Stats:    stats,
		logger:   logger,
	}

	return metrics, nil
}

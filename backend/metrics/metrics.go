package metrics

import (
	"github.com/evanofslack/analogdb/logger"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	AnalogdbNamespace = "analogdb"
	HttpSubsystem     = "http"
	CacheSubsystem    = "cache"
)

// track stats from cache

type Metrics struct {
	Registry *prometheus.Registry
	logger   *logger.Logger
}

func New(logger *logger.Logger) (*Metrics, error) {

	registry := prometheus.NewRegistry()

	metrics := &Metrics{
		Registry: registry,
		logger:   logger,
	}

	return metrics, nil
}

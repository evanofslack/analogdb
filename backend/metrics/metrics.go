package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/evanofslack/analogdb/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const shutdownTimeout = 5 * time.Second

const (
	AnalogdbNamespace = "analogdb"
	HttpSubsystem     = "http"
	CacheSubsystem    = "cache"
)

// track stats from cache

type Metrics struct {
	Registry *prometheus.Registry
	logger   *logger.Logger
	server   *http.Server
}

func New(logger *logger.Logger) (*Metrics, error) {

	logger.Debug().Msg("Created new prometheus registry")

	registry := prometheus.NewRegistry()

	metrics := &Metrics{
		Registry: registry,
		logger:   logger,
	}

	logger.Info().Msg("Initalized prometheus metrics")

	return metrics, nil
}

const metricsPath = "/metrics"

func (m *Metrics) Serve(port string) {
	mux := http.NewServeMux()
	mux.Handle(metricsPath, promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{}))
	addr := ":" + port
	m.server = &http.Server{Addr: addr, Handler: mux}

	m.logger.Info().Msg(fmt.Sprintf("Serving prometheus metrics server at address %s", m.server.Addr))

	go m.server.ListenAndServe()
}

func (m *Metrics) Close() error {

	m.logger.Debug().Msg("Starting prometheus metrics server close")
	defer m.logger.Info().Msg("Closed prometheus metrics server")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return m.server.Shutdown(ctx)
}

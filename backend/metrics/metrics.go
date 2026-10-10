package metrics

import (
	"context"
	"errors"
	"net"
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
	RedisSubsystem    = "redis"
	PostSubsystem     = "post"
	EventsSubsystem   = "events"
	UiEventsSubsystem = "ui_events"
	SearchSubsystem   = "search"
)

// track stats from cache

type Metrics struct {
	Registry *prometheus.Registry
	logger   *logger.Logger
	server   *http.Server
}

func New(logger *logger.Logger) (*Metrics, error) {
	logger.Debug("Created new prometheus registry")

	registry := prometheus.NewRegistry()

	metrics := &Metrics{
		Registry: registry,
		logger:   logger,
	}

	logger.Info("Initalized prometheus metrics")

	return metrics, nil
}

const metricsPath = "/metrics"

func (m *Metrics) Serve(port string) error {
	mux := http.NewServeMux()
	mux.Handle(metricsPath, promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{}))
	addr := ":" + port
	m.server = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	m.logger.Info("Serving prometheus metrics server", "address", ln.Addr().String())

	go func() {
		if err := m.server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			m.logger.Error("Metrics server stopped", "error", err)
		}
	}()
	return nil
}

func (m *Metrics) Close() error {
	m.logger.Debug("Closing prometheus metrics server")
	defer m.logger.Info("Closed prometheus metrics server")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return m.server.Shutdown(ctx)
}

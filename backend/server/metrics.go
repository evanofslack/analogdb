package server

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const metricsPath = "/metrics"

func (s *Server) mountMetricsHandlers() {
	promHandler := promhttp.HandlerFor(s.metrics.Registry, promhttp.HandlerOpts{})
	s.router.Handle(metricsPath, promHandler)
}

package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/evanofslack/analogdb/metrics"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

// track stats from http server
type httpStats struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	requestSize     *prometheus.SummaryVec
	responseSize    *prometheus.SummaryVec
}

func newHttpStats() *httpStats {

	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.HttpSubsystem,
			Name:      "requests_total",
			Help:      "Number of HTTP requests",
		}, []string{"method", "code", "path"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.HttpSubsystem,
			Name:      "request_duration_seconds",
			Help:      "Latencies for HTTP requests",
		},
		[]string{"method", "code", "path"},
	)

	requestSize := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.HttpSubsystem,
			Name:      "request_size_bytes",
			Help:      "Size of HTTP requests",
		},
		[]string{"method", "code", "path"},
	)

	responseSize := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.HttpSubsystem,
			Name:      "response_size_bytes",
			Help:      "Size of HTTP responses",
		},
		[]string{"method", "code", "path"},
	)

	stats := &httpStats{
		requestsTotal:   requestsTotal,
		requestDuration: requestDuration,
		requestSize:     requestSize,
		responseSize:    responseSize,
	}

	return stats
}

func (stats *httpStats) register(registerer prometheus.Registerer) error {
	registerer.MustRegister(stats.requestsTotal)
	registerer.MustRegister(stats.requestDuration)
	registerer.MustRegister(stats.requestSize)
	registerer.MustRegister(stats.responseSize)
	return nil
}

func (server *Server) collectStats(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		defer func() {
			// grab the path
			rctx := chi.RouteContext(r.Context())
			routePattern := strings.Join(rctx.RoutePatterns, "")
			routePattern = strings.Replace(routePattern, "/*/", "/", -1)

			// grab the method and status code
			method := r.Method
			code := http.StatusText(ww.Status())

			// get the request and response size
			requestSize, err := strconv.ParseFloat(r.Header.Get("Content-Length"), 64)
			if err != nil {
				requestSize = 0
			}
			responseSize := float64(ww.BytesWritten())

			// update prom metrics
			server.stats.requestsTotal.WithLabelValues(method, code, routePattern).Inc()
			server.stats.requestDuration.WithLabelValues(method, code, routePattern).Observe(float64(time.Since(start).Nanoseconds()) / 100000000)
			server.stats.requestSize.WithLabelValues(method, code, routePattern).Observe(requestSize)
			server.stats.responseSize.WithLabelValues(method, code, routePattern).Observe(responseSize)
		}()
	})
}

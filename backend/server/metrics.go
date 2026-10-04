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
	requestsTotal      *prometheus.CounterVec
	requestDuration    *prometheus.HistogramVec
	requestSize        *prometheus.SummaryVec
	responseSize       *prometheus.SummaryVec
	postEncodeFailures prometheus.Counter
	rateLimited        prometheus.Counter
	unknownJSONFields  *prometheus.CounterVec
	legacyRequests     *prometheus.CounterVec
	deprecatedParams   *prometheus.CounterVec
	searchRequests     *prometheus.CounterVec
	searchDuration     *prometheus.HistogramVec
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

	postEncodeFailures := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.PostSubsystem,
			Name:      "encode_failures_total",
			Help:      "Number of created posts that failed to encode",
		},
	)

	rateLimited := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.HttpSubsystem,
			Name:      "rate_limited_total",
			Help:      "Number of HTTP requests rejected by the rate limiter",
		},
	)

	unknownJSONFields := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.HttpSubsystem,
			Name:      "unknown_json_fields_total",
			Help:      "Number of request bodies containing unknown JSON fields",
		},
		[]string{"route"},
	)

	legacyRequests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.HttpSubsystem,
			Name:      "legacy_requests_total",
			Help:      "Number of HTTP requests to deprecated unversioned routes",
		},
		[]string{"route", "client"},
	)

	deprecatedParams := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.HttpSubsystem,
			Name:      "deprecated_param_total",
			Help:      "Number of HTTP requests using a deprecated query parameter",
		},
		[]string{"param", "client"},
	)

	searchRequests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.SearchSubsystem,
			Name:      "requests_total",
			Help:      "Number of search requests by kind and result",
		},
		[]string{"kind", "result"},
	)

	searchDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.SearchSubsystem,
			Name:      "duration_seconds",
			Help:      "Latencies for search requests",
		},
		[]string{"kind"},
	)

	stats := &httpStats{
		requestsTotal:      requestsTotal,
		requestDuration:    requestDuration,
		requestSize:        requestSize,
		responseSize:       responseSize,
		postEncodeFailures: postEncodeFailures,
		rateLimited:        rateLimited,
		unknownJSONFields:  unknownJSONFields,
		legacyRequests:     legacyRequests,
		deprecatedParams:   deprecatedParams,
		searchRequests:     searchRequests,
		searchDuration:     searchDuration,
	}

	return stats
}

func (stats *httpStats) register(registerer prometheus.Registerer) error {
	registerer.MustRegister(stats.requestsTotal)
	registerer.MustRegister(stats.requestDuration)
	registerer.MustRegister(stats.requestSize)
	registerer.MustRegister(stats.responseSize)
	registerer.MustRegister(stats.postEncodeFailures)
	registerer.MustRegister(stats.rateLimited)
	registerer.MustRegister(stats.unknownJSONFields)
	registerer.MustRegister(stats.legacyRequests)
	registerer.MustRegister(stats.deprecatedParams)
	registerer.MustRegister(stats.searchRequests)
	registerer.MustRegister(stats.searchDuration)
	return nil
}

func (server *Server) collectStats(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		//nolint:contextcheck // reads route info from the request context
		defer func() {
			// grab the path
			rctx := chi.RouteContext(r.Context())
			routePattern := strings.Join(rctx.RoutePatterns, "")
			routePattern = strings.ReplaceAll(routePattern, "/*/", "/")

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
			server.stats.requestDuration.WithLabelValues(method, code, routePattern).Observe(time.Since(start).Seconds())
			server.stats.requestSize.WithLabelValues(method, code, routePattern).Observe(requestSize)
			server.stats.responseSize.WithLabelValues(method, code, routePattern).Observe(responseSize)
		}()
	})
}

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
	rateLimited        *prometheus.CounterVec
	authRequests       *prometheus.CounterVec
	unknownJSONFields  *prometheus.CounterVec
	searchRequests     *prometheus.CounterVec
	searchDuration     *prometheus.HistogramVec
	searchSlots        *prometheus.GaugeVec
	searchSlotsInUse   *prometheus.GaugeVec
	statsBuildSeconds  prometheus.Histogram
	statsBuildErrors   prometheus.Counter
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

	rateLimited := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.HttpSubsystem,
			Name:      "rate_limited_total",
			Help:      "Number of HTTP requests rejected by the rate limiter",
		},
		[]string{"role"},
	)

	authRequests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.HttpSubsystem,
			Name:      "auth_requests_total",
			Help:      "Number of HTTP requests with basic auth, by resolved role and whether a legacy pair was used",
		},
		[]string{"role", "legacy"},
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

	searchSlots := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.SearchSubsystem,
			Name:      "slots",
			Help:      "Concurrent search slots by kind",
		},
		[]string{"kind"},
	)

	searchSlotsInUse := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.SearchSubsystem,
			Name:      "slots_in_use",
			Help:      "Search slots in use by kind",
		},
		[]string{"kind"},
	)

	statsBuildSeconds := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.StatsSubsystem,
			Name:      "build_seconds",
			Help:      "Duration of stats snapshot builds",
			Buckets:   []float64{1, 5, 10, 20, 30, 60, 120, 300},
		},
	)

	statsBuildErrors := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.StatsSubsystem,
			Name:      "build_errors_total",
			Help:      "Number of failed stats snapshot builds",
		},
	)

	stats := &httpStats{
		requestsTotal:      requestsTotal,
		requestDuration:    requestDuration,
		requestSize:        requestSize,
		responseSize:       responseSize,
		postEncodeFailures: postEncodeFailures,
		rateLimited:        rateLimited,
		authRequests:       authRequests,
		unknownJSONFields:  unknownJSONFields,
		searchRequests:     searchRequests,
		searchDuration:     searchDuration,
		searchSlots:        searchSlots,
		searchSlotsInUse:   searchSlotsInUse,
		statsBuildSeconds:  statsBuildSeconds,
		statsBuildErrors:   statsBuildErrors,
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
	registerer.MustRegister(stats.authRequests)
	registerer.MustRegister(stats.unknownJSONFields)
	registerer.MustRegister(stats.searchRequests)
	registerer.MustRegister(stats.searchDuration)
	registerer.MustRegister(stats.searchSlots)
	registerer.MustRegister(stats.searchSlotsInUse)
	registerer.MustRegister(stats.statsBuildSeconds)
	registerer.MustRegister(stats.statsBuildErrors)
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

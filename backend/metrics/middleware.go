package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func Middleware(metrics *Metrics) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			// start timing
			start := time.Now()

			// wrap and serve
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
				metrics.Stats.RequestsTotal.WithLabelValues(method, code, routePattern).Inc()
				metrics.Stats.RequestDuration.WithLabelValues(method, code, routePattern).Observe(float64(time.Since(start).Nanoseconds()) / 100000000)
				metrics.Stats.RequestSize.WithLabelValues(method, code, routePattern).Observe(requestSize)
				metrics.Stats.ResponseSize.WithLabelValues(method, code, routePattern).Observe(responseSize)
			}()
		}
		return http.HandlerFunc(fn)
	}
}

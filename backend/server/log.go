package server

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func (server *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		ctx := r.Context()

		next.ServeHTTP(ww, r)

		defer func() {
			if rec := recover(); rec != nil {
				err := rec.(error)
				server.logger.Log().
					Stack().
					Err(err).
					Ctx(ctx).
					Bytes("debug_stack", debug.Stack()).
					Msg("Caught error with recoverer")
				http.Error(ww, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}

			// don't log healthcheck requests
			if p := r.URL.Path; p == healthRoute || p == readyRoute {
				return
			}

			authorized := false
			if a := r.Context().Value(authKey); a != nil {
				authorized = true
			}

			// log end request
			server.logger.Info().
				Ctx(ctx).
				Fields(map[string]interface{}{
					"remote_ip":  r.RemoteAddr,
					"path":       r.URL.Path,
					"proto":      r.Proto,
					"method":     r.Method,
					"user_agent": r.Header.Get("User-Agent"),
					"status":     ww.Status(),
					"latency_ms": float64(time.Since(start).Nanoseconds()) / 1000000.0,
					"bytes_in":   r.Header.Get("Content-Length"),
					"bytes_out":  ww.BytesWritten(),
					"authorized": authorized,
				}).
				Msg("Handled request")

			// log query params at debug level
			server.logger.Debug().
				Ctx(ctx).
				Fields(map[string]interface{}{
					"query": r.URL.Query(),
				}).
				Msg("Request query params")
		}()

	})
}

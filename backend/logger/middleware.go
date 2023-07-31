package logger

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func Middleware(logger *Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			start := time.Now()
			log := logger.With().Logger()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			ctx := r.Context()

			defer func() {
				// Recover and record stack traces in case of a panic
				if rec := recover(); rec != nil {
					log.Error().
						Ctx(ctx).
						Str("type", "error").
						Timestamp().
						Interface("recover_info", rec).
						Bytes("debug_stack", debug.Stack()).
						Msg("Log system error")
					http.Error(ww, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}

				// log end request
				log.Info().
					Ctx(ctx).
					Str("type", "access").
					Timestamp().
					Fields(map[string]interface{}{
						"remote_ip":  r.RemoteAddr,
						"url":        r.URL.Path,
						"proto":      r.Proto,
						"method":     r.Method,
						"user_agent": r.Header.Get("User-Agent"),
						"status":     ww.Status(),
						"latency_ms": float64(time.Since(start).Nanoseconds()) / 1000000.0,
						"bytes_in":   r.Header.Get("Content-Length"),
						"bytes_out":  ww.BytesWritten(),
					}).
					Msg("Incoming request")
			}()

			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}

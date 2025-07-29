package server

import (
	"context"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/go-chi/traceid"

	events "github.com/evanofslack/analogdb/internal/gen/proto/analytics/v1"
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

			traceID := traceid.FromContext(r.Context())
			remoteIP := getRealIP(r)
			protocol := getRealProto(r)
			url := r.URL.String()
			path := r.URL.Path
			scheme := r.URL.Scheme
			method := r.Method
			userAgent := r.Header.Get("User-Agent")
			accept := r.Header.Get("Accept")
			status := ww.Status()
			end := time.Now()
			latency := float64(time.Since(start).Milliseconds())
			bytesInStr := r.Header.Get("Content-Length")
			bytesIn := 0
			if bytesInStr != "" {
				var err error
				bytesIn, err = strconv.Atoi(bytesInStr)
				if err != nil {
					server.logger.Warn().Str("bytes_in_str", bytesInStr).Msg("parse content-length to int")
				}
			}
			bytesOut := ww.BytesWritten()

			// log end request
			server.logger.Info().
				Ctx(ctx).
				Fields(map[string]interface{}{
					"trace_id":   traceID,
					"remote_ip":  remoteIP,
					"path":       path,
					"protocol":   protocol,
					"scheme":     scheme,
					"method":     method,
					"user_agent": userAgent,
					"accept":     accept,
					"status":     status,
					"latency_ms": latency,
					"bytes_in":   bytesIn,
					"bytes_out":  bytesOut,
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

			// send request to event stream too
			event := &events.Event{
				RequestId:     traceID,
				RemoteIp:      remoteIP,
				Url:           url,
				Path:          path,
				Protocol:      protocol,
				Scheme:        scheme,
				Method:        method,
				UserAgent:     userAgent,
				ResponseCode:  int32(status),
				Hostname:      server.hostname,
				Authorized:    authorized,
				StartTime:     int64(start.UnixMilli()),
				EndTime:       int64(end.UnixMilli()),
				RequestTimeMs: int64(latency),
				BytesIn:       int32(bytesIn),
				BytesOut:      int32(bytesOut),
			}
			if err := server.EventService.Write(context.Background(), event); err != nil {
				server.logger.Warn().Err(err).Msg("Write request to event stream")
			}
		}()
	})
}

func getRealIP(req *http.Request) string {
	if realIp := req.Header.Get("X-Real-IP"); realIp != "" {
		return realIp
	}
	if remoteIp := req.Header.Get("X-Forwarded-For"); remoteIp != "" {
		return remoteIp
	}
    host, _, _ := net.SplitHostPort(req.RemoteAddr)
    return host
}

func getRealProto(req *http.Request) string {
	if proto := req.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}
    return req.Proto
}

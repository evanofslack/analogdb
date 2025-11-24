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
				server.logger.ErrorContext(ctx, "Caught and recovered", "error", err, "stack", debug.Stack())
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
					server.logger.Warn("Parse content-length to int", "bytes_in_str", bytesInStr)
				}
			}
			bytesOut := ww.BytesWritten()

			// log end request
			server.logger.InfoContext(ctx, "Handle request",
				"trace_id", traceID,
				"remote_ip", remoteIP,
				"path", path,
				"protocol", protocol,
				"scheme", scheme,
				"method", method,
				"user_agent", userAgent,
				"accept", accept,
				"status", status,
				"latency_ms", latency,
				"bytes_in", bytesIn,
				"bytes_out", bytesOut,
				"authorized", authorized,
				"query_params", r.URL.Query(),
			)

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
				server.logger.WarnContext(ctx, "Fail write request to event stream", "error", err)
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

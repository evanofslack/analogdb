package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
)

const (
	maxBodyBytes          = 1 << 20
	errCodeTooLarge       = "too_large"
	unknownFieldErrPrefix = "json: unknown field "
)

func (s *Server) decodeBody(w http.ResponseWriter, r *http.Request, v any, message string) error {
	return s.decodeBodyLimit(w, r, v, message, maxBodyBytes)
}

func (s *Server) decodeBodyLimit(w http.ResponseWriter, r *http.Request, v any, message string, limit int64) error {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limit))
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return &analogdb.Error{Code: errCodeTooLarge, Message: "request body too large"}
		}
		return &analogdb.Error{Code: analogdb.ERRUNPROCESSABLE, Message: message}
	}

	strict := json.NewDecoder(bytes.NewReader(body))
	strict.DisallowUnknownFields()
	probe := reflect.New(reflect.TypeOf(v).Elem()).Interface()
	if err := strict.Decode(probe); err != nil && strings.HasPrefix(err.Error(), unknownFieldErrPrefix) {
		route := "unknown"
		if rctx := chi.RouteContext(r.Context()); rctx != nil {
			route = rctx.RoutePattern()
		}
		field := strings.TrimPrefix(err.Error(), unknownFieldErrPrefix)
		s.logger.WarnContext(r.Context(), "Unknown field in request body", "field", field, "route", route)
		s.stats.unknownJSONFields.WithLabelValues(route).Inc()
	}

	if err := json.NewDecoder(bytes.NewReader(body)).Decode(v); err != nil {
		return &analogdb.Error{Code: analogdb.ERRUNPROCESSABLE, Message: message}
	}
	return nil
}

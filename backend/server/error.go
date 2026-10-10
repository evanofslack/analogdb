package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/evanofslack/analogdb"
)

var codes = map[string]int{
	analogdb.ERRINTERNAL:      http.StatusInternalServerError,
	analogdb.ERRUNPROCESSABLE: http.StatusUnprocessableEntity,
	analogdb.ERRNOTFOUND:      http.StatusNotFound,
	analogdb.ERRUNAVAILABLE:   http.StatusServiceUnavailable,
	analogdb.ERRUNAUTHORIZED:  http.StatusUnauthorized,
	analogdb.ERRFORBIDDEN:     http.StatusForbidden,
	analogdb.ERRBADREQUEST:    http.StatusBadRequest,
	analogdb.ERRCONFLICT:      http.StatusConflict,
	errCodeTooLarge:           http.StatusRequestEntityTooLarge,
	errCodeUnsupportedMedia:   http.StatusUnsupportedMediaType,
}

func errorStatusCode(code string) int {
	if v, ok := codes[code]; ok {
		return v
	}
	return http.StatusInternalServerError
}

func badRequest(format string, a ...any) error {
	return &analogdb.Error{Code: analogdb.ERRBADREQUEST, Message: fmt.Sprintf(format, a...)}
}

func (s *Server) writeError(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()
	code, message := analogdb.ErrorCode(err), analogdb.ErrorMessage(err)
	status := errorStatusCode(code)

	// client errors are expected, keep ERROR for server failures
	level := slog.LevelError
	if status < http.StatusInternalServerError {
		level = slog.LevelInfo
	}
	s.logger.Log(ctx, level, message, "error", err, "method", r.Method, "path", r.URL.Path, "code", code, "status", status)

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(status)
	marshallErr := json.NewEncoder(w).Encode(&ErrorResponse{Error: message})
	if marshallErr != nil {
		s.logger.ErrorContext(ctx, "Fail json marshall server error", "error", err)
	}
}

// ErrorResponse represents the standard error response format
type ErrorResponse struct {
	Error string `json:"error" example:"not found"`
}

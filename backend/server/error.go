package server

import (
	"encoding/json"
	"net/http"

	"github.com/evanofslack/analogdb"
)

var codes = map[string]int{
	analogdb.ERRINTERNAL:      http.StatusInternalServerError,
	analogdb.ERRUNPROCESSABLE: http.StatusUnprocessableEntity,
	analogdb.ERRNOTFOUND:      http.StatusNotFound,
	analogdb.ERRUNAVAILABLE:   http.StatusServiceUnavailable,
	analogdb.ERRUNAUTHORIZED:  http.StatusUnauthorized,
}

func errorStatusCode(code string) int {
	if v, ok := codes[code]; ok {
		return v
	}
	return http.StatusInternalServerError
}

func (s *Server) writeError(w http.ResponseWriter, r *http.Request, err error) {

	ctx := r.Context()

	code, message := analogdb.ErrorCode(err), analogdb.ErrorMessage(err)

	s.logger.Error().Err(err).Ctx(ctx).Str("method", r.Method).Str("path", r.URL.Path).Str("code", code).Msg(message)

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(errorStatusCode(code))
	marshallErr := json.NewEncoder(w).Encode(&ErrorResponse{Error: message})
	if marshallErr != nil {
		s.logger.Error().Err(err).Ctx(ctx).Msg("Failed to marshall json")
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

package server

import (
	"encoding/json"
	"log"
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

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	code, message := analogdb.ErrorCode(err), analogdb.ErrorMessage(err)
	if code == analogdb.ERRINTERNAL {
		log.Printf("http error: %s %s: %s", r.Method, r.URL.Path, err)
	}

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(errorStatusCode(code))
	marshallErr := json.NewEncoder(w).Encode(&ErrorResponse{Error: message})
	if marshallErr != nil {
		log.Fatal(marshallErr)
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}
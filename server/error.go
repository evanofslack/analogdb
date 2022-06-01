package server

import (
	"encoding/json"
	"net/http"
)

const internalError = "Internal error."

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(&ErrorResponse{Error: internalError})
}

type ErrorResponse struct {
	Error string `json:"error"`
}

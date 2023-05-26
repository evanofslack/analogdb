package server

import (
	"encoding/json"
	"net/http"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
)

const (
	encodePath = "/encode"
)

func (s *Server) mountSimilarityHandlers() {
	s.router.Route(encodePath, func(r chi.Router) {
		r.With(auth).Put("/", s.encodePosts)
	})
}

type encodePostsRequest struct {
	Ids       []int `json:"ids"`
	BatchSize int   `json:"batch_size"`
}

type encodePostsResponse struct {
	Message string `json:"message"`
}

func (s *Server) encodePosts(w http.ResponseWriter, r *http.Request) {

	var request encodePostsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		err = &analogdb.Error{Code: analogdb.ERRUNPROCESSABLE, Message: "error parsing ids or batch_size from request body"}
		writeError(w, r, err)
	}

	err := s.SimilarityService.BatchEncodePosts(r.Context(), request.Ids, request.BatchSize)
	if err != nil {
		writeError(w, r, err)
	}
	response := encodePostsResponse{
		Message: "successfully encoded posts",
	}
	if err := encodeResponse(w, r, http.StatusOK, response); err != nil {
		writeError(w, r, err)
	}
}

package server

import (
	"encoding/json"
	"fmt"
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
		s.writeError(w, r, err)
	}

	var message string

	// encode single post
	if len(request.Ids) == 1 {
		err := s.SimilarityService.EncodePost(r.Context(), request.Ids[0])
		if err != nil {
			s.writeError(w, r, err)
		}
		message = "successfully encoded post"

	} else {
		// encode batch of posts
		err := s.SimilarityService.BatchEncodePosts(r.Context(), request.Ids, request.BatchSize)
		if err != nil {
			s.writeError(w, r, err)
		}
		message = fmt.Sprintf("successfully encoded %d posts", len(request.Ids))
	}

	response := encodePostsResponse{
		Message: message,
	}
	if err := encodeResponse(w, r, http.StatusOK, response); err != nil {
		s.writeError(w, r, err)
	}
}

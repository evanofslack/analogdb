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

func (s *Server) mountSimilarityHandlers(r chi.Router) {
	r.Route(encodePath, func(r chi.Router) {
		r.With(s.auth).Put("/", s.encodePosts)
	})
}

type encodePostsRequest struct {
    Ids       []int `json:"ids" examples:"1,2,3,4,5"`
    BatchSize int   `json:"batch_size" example:"20"`
}

type encodePostsResponse struct {
    Message string `json:"message" example:"successfully encoded 5 posts"`
}

// @Summary Encode posts for similarity matching
// @Description Encode posts to generate embeddings for similarity search. Supports both single post and batch encoding (requires authentication)
// @Tags similarity
// @Accept json
// @Produce json
// @Param request body encodePostsRequest true "Post IDs and batch size for encoding"
// @Success 200 {object} encodePostsResponse
// @Failure 400 {object} analogdb.Error "Invalid request body"
// @Failure 401 {object} analogdb.Error "Unauthorized"
// @Failure 422 {object} analogdb.Error "Unprocessable entity"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Security BasicAuth
// @Router /encode [put]
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

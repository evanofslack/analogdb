package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/evanofslack/analogdb"
)

const (
	hashesPath             = scrapePath + "/hashes"
	hashesMissingPath      = hashesPath + "/missing"
	duplicatesPath         = scrapePath + "/duplicates"
	maxHashesPerPut        = 500
	duplicatesWriteTimeout = 5 * time.Minute
)

type HashesMissingResponse struct {
	Posts []analogdb.ImageHashMissing `json:"posts"`
}

type SetHashesRequest struct {
	Hashes []analogdb.ImageHash `json:"hashes"`
}

type SetHashesResponse struct {
	Updated int `json:"updated" example:"200"`
}

type DuplicatesResponse struct {
	Duplicates []analogdb.DuplicatePost `json:"duplicates"`
}

// @Summary List posts missing an image hash
// @Description List posts with no image hash and their low res image url, oldest id first (requires authentication)
// @Tags scrape
// @Produce json
// @Success 200 {object} HashesMissingResponse
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /scrape/hashes/missing [get]
func (s *Server) getHashMissingPosts(w http.ResponseWriter, r *http.Request) {
	posts, err := s.ScrapeService.ImageHashMissingPosts(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := encodeResponse(w, r, http.StatusOK, HashesMissingResponse{Posts: posts}); err != nil {
		s.writeError(w, r, err)
	}
}

// @Summary Set image hashes
// @Description Store image hashes for up to 500 posts, without changing their updated time (requires authentication)
// @Tags scrape
// @Accept json
// @Produce json
// @Param hashes body SetHashesRequest true "Hashes as 16 hex characters"
// @Success 200 {object} SetHashesResponse
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /scrape/hashes [put]
func (s *Server) putImageHashes(w http.ResponseWriter, r *http.Request) {
	var req SetHashesRequest
	if err := s.decodeBody(w, r, &req, "parse hashes from request body"); err != nil {
		s.writeError(w, r, err)
		return
	}
	if len(req.Hashes) > maxHashesPerPut {
		s.writeError(w, r, badRequest("too many hashes, max %d per request", maxHashesPerPut))
		return
	}
	for _, h := range req.Hashes {
		if _, err := analogdb.ParseImageHash(h.Hash); err != nil {
			s.writeError(w, r, err)
			return
		}
	}
	updated, err := s.ScrapeService.SetImageHashes(r.Context(), req.Hashes)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := encodeResponse(w, r, http.StatusOK, SetHashesResponse{Updated: updated}); err != nil {
		s.writeError(w, r, err)
	}
}

// @Summary List duplicate posts
// @Description List every post whose image hash is within DUPLICATE_MAX_BITS of an older post, paired with the oldest match. Can take seconds (requires authentication)
// @Tags scrape
// @Produce json
// @Success 200 {object} DuplicatesResponse
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /scrape/duplicates [get]
func (s *Server) getDuplicatePosts(w http.ResponseWriter, r *http.Request) {
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(duplicatesWriteTimeout)); err != nil {
		s.logger.WarnContext(r.Context(), "Fail extend write deadline for duplicates", "error", err)
	}

	dupes, err := s.ScrapeService.DuplicatePosts(r.Context(), s.config.App.DuplicateMaxBits)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := encodeResponse(w, r, http.StatusOK, DuplicatesResponse{Duplicates: dupes}); err != nil {
		s.writeError(w, r, err)
	}
}

// checkDuplicate looks for an existing post with the same image. A match is
// logged, and rejected when blocking is on. A failed lookup never blocks the
// upload.
func (s *Server) checkDuplicate(ctx context.Context, post *analogdb.CreatePost) error {
	if post.ImageHash == nil {
		return nil
	}
	hash, err := analogdb.ParseImageHash(*post.ImageHash)
	if err != nil {
		return err
	}
	id, found, err := s.ScrapeService.FindImageMatch(ctx, hash, post.Grayscale, s.config.App.DuplicateMaxBits)
	if err != nil {
		s.logger.ErrorContext(ctx, "Fail find duplicate post", "permalink", post.Permalink, "error", err)
		return nil
	}
	if !found {
		return nil
	}
	block := s.config.App.DuplicateBlock
	s.logger.InfoContext(ctx, "Found duplicate post", "permalink", post.Permalink, "match_id", id, "blocked", block)
	if block {
		return &analogdb.Error{Code: analogdb.ERRCONFLICT, Message: fmt.Sprintf("post is a duplicate of post %d", id)}
	}
	return nil
}

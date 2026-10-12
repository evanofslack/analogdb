package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/config"
)

const (
	duplicatesPath         = scrapePath + "/duplicates"
	duplicatesWriteTimeout = 5 * time.Minute
)

type DuplicatesResponse struct {
	Duplicates []analogdb.DuplicatePost `json:"duplicates"`
}

type ReportDuplicatesResponse struct {
	Duplicates []analogdb.DuplicatePost `json:"duplicates"`
	Reported   int                      `json:"reported" example:"12"`
}

// @Summary List duplicate posts
// @Description List every post that duplicates an older post, paired with the oldest match. Hashes within 2 bits match from any author and within 4 bits from the same author, with aspect ratios within 5%. Can take seconds (requires authentication)
// @Tags scrape
// @Produce json
// @Success 200 {object} DuplicatesResponse
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /scrape/duplicates [get]
func (s *Server) getDuplicatePosts(w http.ResponseWriter, r *http.Request) {
	s.extendDuplicatesDeadline(w, r)

	dupes, err := s.ScrapeService.DuplicatePosts(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := encodeResponse(w, r, http.StatusOK, DuplicatesResponse{Duplicates: dupes}); err != nil {
		s.writeError(w, r, err)
	}
}

// @Summary Report duplicate posts
// @Description Find duplicate posts as GET /scrape/duplicates does and file a duplicate report on each one without an open report. Nothing is deleted. Can take seconds (requires authentication)
// @Tags scrape
// @Produce json
// @Success 200 {object} ReportDuplicatesResponse
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /scrape/duplicates/report [post]
func (s *Server) reportDuplicatePosts(w http.ResponseWriter, r *http.Request) {
	s.extendDuplicatesDeadline(w, r)

	dupes, err := s.ScrapeService.DuplicatePosts(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	reported, err := s.ScrapeService.ReportDuplicates(r.Context(), dupes)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := encodeResponse(w, r, http.StatusOK, ReportDuplicatesResponse{Duplicates: dupes, Reported: reported}); err != nil {
		s.writeError(w, r, err)
	}
}

func (s *Server) extendDuplicatesDeadline(w http.ResponseWriter, r *http.Request) {
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(duplicatesWriteTimeout)); err != nil {
		s.logger.WarnContext(r.Context(), "Fail extend write deadline for duplicates", "error", err)
	}
}

// checkDuplicate looks for an existing post with the same image and returns
// it, or rejects the upload in block mode. A failed lookup never blocks the
// upload.
func (s *Server) checkDuplicate(ctx context.Context, post *analogdb.CreatePost) (*analogdb.ImageMatch, error) {
	mode := s.config.App.DuplicateMode
	if post.ImageHash == nil || mode == "" || mode == config.DuplicateOff {
		return nil, nil
	}
	hash, err := analogdb.ParseImageHash(*post.ImageHash)
	if err != nil {
		return nil, err
	}
	if len(post.Images) != 4 {
		return nil, nil
	}
	raw := post.Images[3]
	match, found, err := s.ScrapeService.FindImageMatch(ctx, analogdb.ImageMatchQuery{
		Hash:      hash,
		Grayscale: post.Grayscale,
		Author:    post.Author,
		Width:     raw.Width,
		Height:    raw.Height,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "Fail find duplicate post", "permalink", post.Permalink, "error", err)
		return nil, nil
	}
	if !found {
		return nil, nil
	}
	s.logger.InfoContext(ctx, "Found duplicate post", "permalink", post.Permalink, "match_id", match.ID, "distance", match.Distance, "mode", mode)
	if mode == config.DuplicateBlock {
		return nil, &analogdb.Error{Code: analogdb.ERRCONFLICT, Message: fmt.Sprintf("post is a duplicate of post %d", match.ID)}
	}
	return &match, nil
}

// reportDuplicate files a duplicate report on a new post in report mode. A
// failed report is only logged, since the post is already created.
func (s *Server) reportDuplicate(ctx context.Context, match *analogdb.ImageMatch, postID int) {
	if match == nil || s.config.App.DuplicateMode != config.DuplicateReport {
		return
	}
	dupe := analogdb.DuplicatePost{KeepID: match.ID, DuplicateID: postID, Distance: match.Distance, SameAuthor: match.SameAuthor}
	if _, err := s.ScrapeService.ReportDuplicates(ctx, []analogdb.DuplicatePost{dupe}); err != nil {
		s.logger.ErrorContext(ctx, "Fail report duplicate post", "post_id", postID, "match_id", match.ID, "error", err)
	}
}

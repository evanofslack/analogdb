package server

import (
	"context"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
)

const (
	errCodeTooManyReports = "too_many_reports"
	maxReportBodyBytes    = 4 << 10
	maxReportMessage      = 500
	maxReportEmail        = 254
	reportRateLimit       = 5
	reportRateLimitAfter  = time.Hour
	defaultReportsLimit   = 50
	maxReportsLimit       = 200
)

type CreateReportResponse struct {
	ID int `json:"id" example:"12"`
}

type RemovedPermalinksResponse struct {
	Permalinks []string `json:"permalinks" example:"/r/analog/comments/abc123/dusk/"`
}

type adminReportsResponse struct {
	Reports    []*analogdb.Report `json:"reports"`
	NextBefore *int               `json:"next_before"`
}

// reportLimiter caps reports per client IP on top of the global limits, since
// each one lands in the admin queue. Admin is never limited, and web limits
// per visitor itself because all its requests share one IP here.
func (s *Server) reportLimiter() func(http.Handler) http.Handler {
	if !s.config.App.RateLimitEnabled {
		return func(next http.Handler) http.Handler { return next }
	}
	limit := httprate.LimitBy(reportRateLimit, reportRateLimitAfter, keyByClientIP,
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			s.stats.rateLimited.WithLabelValues("report").Inc()
			s.writeError(w, r, &analogdb.Error{Code: errCodeTooManyReports, Message: "too many reports, try again later"})
		}))
	return func(next http.Handler) http.Handler {
		limited := limit(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if p := principalFrom(r); p != nil && (p.role == roleAdmin || p.role == roleWeb) {
				next.ServeHTTP(w, r)
				return
			}
			limited.ServeHTTP(w, r)
		})
	}
}

// @Summary Report a post
// @Description Flag a post for the admin, for a takedown request, a wrong NSFW label, wrong info or a photo that isn't film
// @Tags post
// @Accept json
// @Produce json
// @Param id path int true "Post ID"
// @Param report body analogdb.CreateReport true "Report"
// @Success 201 {object} CreateReportResponse
// @Failure 400 {object} ErrorResponse "Invalid report"
// @Failure 404 {object} ErrorResponse "Post not found"
// @Failure 429 {object} ErrorResponse "Too many reports"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /post/{id}/report [post]
func (s *Server) createReport(w http.ResponseWriter, r *http.Request) {
	id, err := stringToInt(chi.URLParam(r, "id"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}

	var report analogdb.CreateReport
	if err := s.decodeBodyLimit(w, r, &report, "parse report from request body", maxReportBodyBytes); err != nil {
		s.writeError(w, r, err)
		return
	}
	report.PostID = id
	if err := validateReport(&report); err != nil {
		s.writeError(w, r, err)
		return
	}

	reportID, err := s.ReportService.CreateReport(r.Context(), &report)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := encodeResponse(w, r, http.StatusCreated, CreateReportResponse{ID: reportID}); err != nil {
		s.writeError(w, r, err)
	}
}

// validateReport trims the text fields in place and checks them
func validateReport(report *analogdb.CreateReport) error {
	if !report.Reason.Valid() {
		return badRequest("invalid reason %q", report.Reason)
	}
	report.Message = strings.TrimSpace(report.Message)
	if utf8.RuneCountInString(report.Message) > maxReportMessage {
		return badRequest("message is too long, max %d characters", maxReportMessage)
	}
	report.Email = strings.TrimSpace(report.Email)
	if report.Email != "" {
		if len(report.Email) > maxReportEmail {
			return badRequest("email is too long")
		}
		if addr, err := mail.ParseAddress(report.Email); err != nil || addr.Address != report.Email {
			return badRequest("invalid email")
		}
	}
	return nil
}

func (s *Server) mountAdminReportHandlers(r chi.Router) {
	r.Get("/reports", s.getAdminReports)
	r.Post("/reports/{id}/resolve", s.resolveReport)
	r.Post("/reports/{id}/takedown", s.takedownReport)
}

func (s *Server) getAdminReports(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	filter := &analogdb.ReportFilter{Status: analogdb.ReportStatusOpen, Limit: defaultReportsLimit}
	if str := query.Get("status"); str != "" {
		status := analogdb.ReportStatus(str)
		switch status {
		case analogdb.ReportStatusOpen, analogdb.ReportStatusResolved, analogdb.ReportStatusAll:
			filter.Status = status
		default:
			s.writeError(w, r, badRequest("invalid status %q, want open, resolved or all", str))
			return
		}
	}
	if str := query.Get("limit"); str != "" {
		val, err := stringToInt(str)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		filter.Limit = clampLimit(val, defaultReportsLimit, 1, maxReportsLimit)
	}
	if str := query.Get("before"); str != "" {
		val, err := stringToInt(str)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		filter.BeforeID = &val
	}

	reports, err := s.ReportService.FindReports(r.Context(), filter)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	response := adminReportsResponse{Reports: reports}
	if len(reports) == filter.Limit {
		next := reports[len(reports)-1].ID
		response.NextBefore = &next
	}
	if err := encodeResponse(w, r, http.StatusOK, response); err != nil {
		s.writeError(w, r, err)
	}
}

func (s *Server) resolveReport(w http.ResponseWriter, r *http.Request) {
	id, err := stringToInt(chi.URLParam(r, "id"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := s.ReportService.ResolveReport(r.Context(), id); err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := encodeResponse(w, r, http.StatusOK, map[string]string{"message": "Success, report resolved"}); err != nil {
		s.writeError(w, r, err)
	}
}

// takedownReport deletes the reported post and resolves every open report on
// it. A post that is already gone still has its reports resolved, so a failed
// takedown can be retried.
func (s *Server) takedownReport(w http.ResponseWriter, r *http.Request) {
	id, err := stringToInt(chi.URLParam(r, "id"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	report, err := s.ReportService.FindReportByID(r.Context(), id)
	if err != nil {
		s.writeError(w, r, err)
		return
	}

	err = s.PostService.DeletePost(r.Context(), report.PostID, string(report.Reason))
	if err != nil && !(analogdb.ErrorCode(err) == analogdb.ERRNOTFOUND && report.Post.Removed) {
		s.writeError(w, r, err)
		return
	}
	if err := s.deleteVector(r.Context(), report.PostID); err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := s.ReportService.ResolvePostReports(r.Context(), report.PostID); err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := encodeResponse(w, r, http.StatusOK, DeleteResponse{Message: "Success, post deleted"}); err != nil {
		s.writeError(w, r, err)
	}
}

// deleteVector removes a deleted post from the vector DB. A post that was never
// encoded, or whose vector is already gone, is fine.
func (s *Server) deleteVector(ctx context.Context, id int) error {
	err := s.SimilarityService.DeletePost(ctx, id)
	if analogdb.ErrorCode(err) == analogdb.ERRNOTFOUND {
		return nil
	}
	return err
}

// @Summary List removed post permalinks
// @Description Permalinks of every deleted post, so the scraper never adds them again (requires authentication)
// @Tags removed
// @Produce json
// @Success 200 {object} RemovedPermalinksResponse
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /admin/removed/permalinks [get]
func (s *Server) getRemovedPermalinks(w http.ResponseWriter, r *http.Request) {
	permalinks, err := s.ReportService.RemovedPermalinks(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := encodeResponse(w, r, http.StatusOK, RemovedPermalinksResponse{Permalinks: permalinks}); err != nil {
		s.writeError(w, r, err)
	}
}

package analogdb

import (
	"context"
	"time"
)

// ReportReason is why a visitor flagged a post. Kept as text in postgres so
// new reasons only need a change here.
type ReportReason string

const (
	ReportTakedown       ReportReason = "takedown"
	ReportNsfwMislabeled ReportReason = "nsfw_mislabeled"
	ReportWrongInfo      ReportReason = "wrong_info"
	ReportNotFilm        ReportReason = "not_film"
	ReportOther          ReportReason = "other"
)

var ReportReasons = []ReportReason{
	ReportTakedown,
	ReportNsfwMislabeled,
	ReportWrongInfo,
	ReportNotFilm,
	ReportOther,
}

func (r ReportReason) Valid() bool {
	for _, reason := range ReportReasons {
		if r == reason {
			return true
		}
	}
	return false
}

type ReportStatus string

const (
	ReportStatusOpen     ReportStatus = "open"
	ReportStatusResolved ReportStatus = "resolved"
	ReportStatusAll      ReportStatus = "all"
)

type CreateReport struct {
	PostID  int          `json:"-"`
	Reason  ReportReason `json:"reason" example:"takedown" enums:"takedown,nsfw_mislabeled,wrong_info,not_film,other"`
	Message string       `json:"message,omitempty" example:"This is my photo, please remove it"`
	Email   string       `json:"email,omitempty" example:"me@example.com"`
}

// ReportPost is a snapshot of the reported post, from pictures while it
// exists and from removed_posts after a takedown
type ReportPost struct {
	Title     *string `json:"title"`
	Author    *string `json:"author"`
	Permalink *string `json:"permalink"`
	LowURL    *string `json:"low_url"`
	Nsfw      *bool   `json:"nsfw"`
	Removed   bool    `json:"removed"`
}

type Report struct {
	ID         int          `json:"id"`
	PostID     int          `json:"post_id"`
	Reason     ReportReason `json:"reason"`
	Message    *string      `json:"message"`
	Email      *string      `json:"email"`
	CreatedAt  time.Time    `json:"created_at"`
	ResolvedAt *time.Time   `json:"resolved_at"`
	Post       ReportPost   `json:"post"`
}

type ReportFilter struct {
	Status   ReportStatus
	BeforeID *int
	Limit    int
}

type ReportService interface {
	CreateReport(ctx context.Context, report *CreateReport) (int, error)
	FindReports(ctx context.Context, filter *ReportFilter) ([]*Report, error)
	FindReportByID(ctx context.Context, id int) (*Report, error)
	// ResolveReport marks an open report resolved and drops its email
	ResolveReport(ctx context.Context, id int) error
	// ResolvePostReports resolves every open report on a post
	ResolvePostReports(ctx context.Context, postID int) error
	// RemovedPermalinks lists the permalinks of every deleted post
	RemovedPermalinks(ctx context.Context) ([]string, error)
}

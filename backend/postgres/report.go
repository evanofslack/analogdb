package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/evanofslack/analogdb"
)

// ensure interface is implemented
var _ analogdb.ReportService = (*ReportService)(nil)

type ReportService struct {
	db *DB
}

func NewReportService(db *DB) *ReportService {
	return &ReportService{db: db}
}

func (s *ReportService) CreateReport(ctx context.Context, report *analogdb.CreateReport) (int, error) {
	s.db.logger.DebugContext(ctx, "Starting create report", "post_id", report.PostID)
	defer s.db.logger.DebugContext(ctx, "Finished create report")

	// only live posts can be reported, the select inserts nothing otherwise
	query := `
		INSERT INTO post_reports (post_id, reason, message, email)
		SELECT id, $2, NULLIF($3, ''), NULLIF($4, '')
		FROM pictures
		WHERE id = $1
		RETURNING id`

	var id int
	err := s.db.db.QueryRowContext(ctx, query, report.PostID, string(report.Reason), report.Message, report.Email).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "post not found"}
	}
	if err != nil {
		s.db.logger.ErrorContext(ctx, "Fail create report", "post_id", report.PostID, "error", err)
		return 0, err
	}
	s.db.logger.InfoContext(ctx, "Created report", "report_id", id, "post_id", report.PostID, "reason", report.Reason)
	return id, nil
}

// reportPostColumns snapshots a post from pictures while it exists and from
// removed_posts after a takedown
func reportPostColumns(p, rp string) string {
	return fmt.Sprintf(`
		COALESCE(%[1]s.title, %[2]s.title), COALESCE(%[1]s.author, %[2]s.author),
		COALESCE(%[1]s.permalink, %[2]s.permalink), COALESCE(%[1]s.lowurl, %[2]s.low_url),
		COALESCE(%[1]s.medurl, %[2]s.med_url), COALESCE(to_timestamp(%[1]s.time), %[2]s.posted_at),
		%[1]s.width, %[1]s.height, %[1]s.nsfw, %[1]s.id IS NULL AND %[2]s.post_id IS NOT NULL`, p, rp)
}

var reportSelect = `
	SELECT r.id, r.post_id, r.reason, r.message, r.email, r.created_at, r.resolved_at,
		r.duplicate_of,` + reportPostColumns("p", "rp") + `,` + reportPostColumns("op", "orp") + `
	FROM post_reports r
	LEFT JOIN pictures p ON p.id = r.post_id
	LEFT JOIN removed_posts rp ON rp.post_id = r.post_id
	LEFT JOIN pictures op ON op.id = r.duplicate_of
	LEFT JOIN removed_posts orp ON orp.post_id = r.duplicate_of`

func (s *ReportService) FindReports(ctx context.Context, filter *analogdb.ReportFilter) ([]*analogdb.Report, error) {
	s.db.logger.DebugContext(ctx, "Starting find reports")
	defer s.db.logger.DebugContext(ctx, "Finished find reports")

	where, args := []string{}, []any{}
	if filter.Reason != "" {
		args = append(args, string(filter.Reason))
		where = append(where, fmt.Sprintf("r.reason = $%d", len(args)))
	}
	switch filter.Status {
	case analogdb.ReportStatusOpen:
		where = append(where, "r.resolved_at IS NULL")
	case analogdb.ReportStatusResolved:
		where = append(where, "r.resolved_at IS NOT NULL")
	}
	if filter.BeforeID != nil {
		args = append(args, *filter.BeforeID)
		where = append(where, fmt.Sprintf("r.id < $%d", len(args)))
	}

	query := reportSelect
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	args = append(args, filter.Limit)
	query += fmt.Sprintf(" ORDER BY r.id DESC LIMIT $%d", len(args))

	rows, err := s.db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reports := make([]*analogdb.Report, 0)
	for rows.Next() {
		report, err := scanReport(rows, s.db.imageBase)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, rows.Err()
}

func (s *ReportService) FindReportByID(ctx context.Context, id int) (*analogdb.Report, error) {
	report, err := scanReport(s.db.db.QueryRowContext(ctx, reportSelect+" WHERE r.id = $1", id), s.db.imageBase)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "report not found"}
	}
	return report, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanReport(row rowScanner, imageBase string) (*analogdb.Report, error) {
	var r analogdb.Report
	var reason string
	var message, email sql.NullString
	var resolvedAt sql.NullTime
	var duplicateOf sql.NullInt64
	var post, original reportPostScan
	dest := []any{&r.ID, &r.PostID, &reason, &message, &email, &r.CreatedAt, &resolvedAt, &duplicateOf}
	dest = append(dest, post.dest()...)
	dest = append(dest, original.dest()...)
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	r.Reason = analogdb.ReportReason(reason)
	r.Message = nullStringPtr(message)
	r.Email = nullStringPtr(email)
	if resolvedAt.Valid {
		r.ResolvedAt = &resolvedAt.Time
	}
	r.Post = post.reportPost(imageBase)
	if duplicateOf.Valid {
		id := int(duplicateOf.Int64)
		r.DuplicateOf = &id
		o := original.reportPost(imageBase)
		r.Original = &o
	}
	return &r, nil
}

// reportPostScan holds the columns of one reportPostColumns snapshot
type reportPostScan struct {
	title, author, permalink, lowURL, medURL sql.NullString
	postedAt                                 sql.NullTime
	width, height                            sql.NullInt64
	nsfw                                     sql.NullBool
	removed                                  bool
}

func (s *reportPostScan) dest() []any {
	return []any{&s.title, &s.author, &s.permalink, &s.lowURL, &s.medURL, &s.postedAt,
		&s.width, &s.height, &s.nsfw, &s.removed}
}

func (s *reportPostScan) reportPost(imageBase string) analogdb.ReportPost {
	post := analogdb.ReportPost{
		Title:     nullStringPtr(s.title),
		Author:    nullStringPtr(s.author),
		Permalink: nullStringPtr(s.permalink),
		LowURL:    nullImageURL(imageBase, s.lowURL),
		MedURL:    nullImageURL(imageBase, s.medURL),
		Width:     nullIntPtr(s.width),
		Height:    nullIntPtr(s.height),
		Removed:   s.removed,
	}
	if s.postedAt.Valid {
		post.PostedAt = &s.postedAt.Time
	}
	if s.nsfw.Valid {
		post.Nsfw = &s.nsfw.Bool
	}
	return post
}

func nullImageURL(imageBase string, n sql.NullString) *string {
	if !n.Valid {
		return nil
	}
	url := imageURL(imageBase, n.String)
	return &url
}

func nullIntPtr(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

func (s *ReportService) ResolveReport(ctx context.Context, id int) error {
	// resolving twice is fine, only a missing report is an error
	query := `
		UPDATE post_reports
		SET resolved_at = COALESCE(resolved_at, NOW()), email = NULL
		WHERE id = $1
		RETURNING id`
	var returned int
	err := s.db.db.QueryRowContext(ctx, query, id).Scan(&returned)
	if errors.Is(err, sql.ErrNoRows) {
		return &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "report not found"}
	}
	if err != nil {
		s.db.logger.ErrorContext(ctx, "Fail resolve report", "report_id", id, "error", err)
		return err
	}
	s.db.logger.InfoContext(ctx, "Resolved report", "report_id", id)
	return nil
}

func (s *ReportService) ResolvePostReports(ctx context.Context, postID int) error {
	query := `
		UPDATE post_reports
		SET resolved_at = NOW(), email = NULL
		WHERE post_id = $1 AND resolved_at IS NULL`
	res, err := s.db.db.ExecContext(ctx, query, postID)
	if err != nil {
		s.db.logger.ErrorContext(ctx, "Fail resolve post reports", "post_id", postID, "error", err)
		return err
	}
	count, _ := res.RowsAffected()
	s.db.logger.InfoContext(ctx, "Resolved post reports", "post_id", postID, "count", count)
	return nil
}

func (s *ReportService) RemovedPermalinks(ctx context.Context) ([]string, error) {
	rows, err := s.db.db.QueryContext(ctx, `SELECT permalink FROM removed_posts ORDER BY post_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	permalinks := make([]string, 0)
	for rows.Next() {
		var permalink string
		if err := rows.Scan(&permalink); err != nil {
			return nil, err
		}
		permalinks = append(permalinks, permalink)
	}
	return permalinks, rows.Err()
}

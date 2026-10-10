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

const reportSelect = `
	SELECT r.id, r.post_id, r.reason, r.message, r.email, r.created_at, r.resolved_at,
		COALESCE(p.title, rp.title), COALESCE(p.author, rp.author),
		COALESCE(p.permalink, rp.permalink), COALESCE(p.lowurl, rp.low_url),
		p.nsfw, p.id IS NULL AND rp.post_id IS NOT NULL
	FROM post_reports r
	LEFT JOIN pictures p ON p.id = r.post_id
	LEFT JOIN removed_posts rp ON rp.post_id = r.post_id`

func (s *ReportService) FindReports(ctx context.Context, filter *analogdb.ReportFilter) ([]*analogdb.Report, error) {
	s.db.logger.DebugContext(ctx, "Starting find reports")
	defer s.db.logger.DebugContext(ctx, "Finished find reports")

	where, args := []string{}, []any{}
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
		report, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, rows.Err()
}

func (s *ReportService) FindReportByID(ctx context.Context, id int) (*analogdb.Report, error) {
	report, err := scanReport(s.db.db.QueryRowContext(ctx, reportSelect+" WHERE r.id = $1", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "report not found"}
	}
	return report, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanReport(row rowScanner) (*analogdb.Report, error) {
	var r analogdb.Report
	var reason string
	var message, email, title, author, permalink, lowURL sql.NullString
	var resolvedAt sql.NullTime
	var nsfw sql.NullBool
	if err := row.Scan(&r.ID, &r.PostID, &reason, &message, &email, &r.CreatedAt, &resolvedAt,
		&title, &author, &permalink, &lowURL, &nsfw, &r.Post.Removed); err != nil {
		return nil, err
	}
	r.Reason = analogdb.ReportReason(reason)
	r.Message = nullStringPtr(message)
	r.Email = nullStringPtr(email)
	if resolvedAt.Valid {
		r.ResolvedAt = &resolvedAt.Time
	}
	r.Post.Title = nullStringPtr(title)
	r.Post.Author = nullStringPtr(author)
	r.Post.Permalink = nullStringPtr(permalink)
	r.Post.LowURL = nullStringPtr(lowURL)
	if nsfw.Valid {
		r.Post.Nsfw = &nsfw.Bool
	}
	return &r, nil
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

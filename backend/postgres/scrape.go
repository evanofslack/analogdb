package postgres

import (
	"context"
	"database/sql"

	"github.com/evanofslack/analogdb"
)

// ensure interface is implemented
var _ analogdb.ScrapeService = (*ScrapeService)(nil)

type ScrapeService struct {
	db *DB
}

func NewScrapeService(db *DB) *ScrapeService {
	return &ScrapeService{db: db}
}

func (s *ScrapeService) KeywordUpdatedPostIDs(ctx context.Context) ([]int, error) {
	s.db.logger.DebugContext(ctx, "Start get keyword updated post ids")
	defer s.db.logger.DebugContext(ctx, "Finish get keyword updated post ids")

	ids, err := keywordUpdatedPostIDs(ctx, s.db.db)
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func (s *ScrapeService) CaptionMissingPostIDs(ctx context.Context, version *string) ([]int, error) {
	s.db.logger.DebugContext(ctx, "Start get caption missing post ids")
	defer s.db.logger.DebugContext(ctx, "Finish get caption missing post ids")

	query := `
			SELECT p.id FROM pictures p
			LEFT JOIN post_captions c ON c.post_id = p.id
			WHERE c.post_id IS NULL OR c.version <> $1
			ORDER BY p.id ASC`
	rows, err := s.db.db.QueryContext(ctx, query, NewNullStringFromPtr(version).ToSQLNullString())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int, 0)
	var id int
	for rows.Next() {
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func keywordUpdatedPostIDs(ctx context.Context, db *sql.DB) ([]int, error) {
	query := `
			SELECT DISTINCT post_id FROM post_updates WHERE keywords_update_time IS NOT NULL ORDER BY post_id ASC`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int, 0)
	var id int
	for rows.Next() {
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

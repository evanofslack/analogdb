package postgres

import (
	"context"

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

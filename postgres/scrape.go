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
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	ids, err := keywordUpdatedPostIDs(ctx, tx)

	if err != nil {
		return nil, err
	}

	return ids, nil
}

func keywordUpdatedPostIDs(ctx context.Context, tx *sql.Tx) ([]int, error) {
	query := `
			SELECT post_id FROM post_updates WHERE keywords_update_time IS NOT NULL ORDER BY post_id ASC`
	rows, err := tx.QueryContext(ctx, query)
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
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return ids, nil
}

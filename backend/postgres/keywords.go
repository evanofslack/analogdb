package postgres

import (
	"context"
	"database/sql"

	"github.com/evanofslack/analogdb"
)

// ensure interface is implemented
var _ analogdb.KeywordService = (*KeywordService)(nil)

type KeywordService struct {
	db *DB
}

func NewKeywordService(db *DB) *KeywordService {
	return &KeywordService{db: db}
}

func (s *KeywordService) GetKeywordSummary(ctx context.Context, limit int) (*[]analogdb.KeywordSummary, error) {

	s.db.logger.Debug().Ctx(ctx).Msg("Starting find keyword summary")
	defer s.db.logger.Debug().Ctx(ctx).Msg("Finished find keyword summary")

	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	summary, err := getKeywordSummary(ctx, tx, limit)

	if err != nil {
		return nil, err
	}

	return summary, nil
}

func getKeywordSummary(ctx context.Context, tx *sql.Tx, limit int) (*[]analogdb.KeywordSummary, error) {
	query := `
			SELECT
				word,
				avg(weight) as avg_weight,
				count(word) as count,
				COUNT(*) OVER() as total
			FROM keywords
			GROUP BY word
			ORDER BY count DESC
			LIMIT $1
	`

	arg := limit

	rows, err := tx.QueryContext(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keywords := make([]analogdb.KeywordSummary, 0)
	var kw analogdb.KeywordSummary
	var total int
	for rows.Next() {
		if err := rows.Scan(&kw.Word, &kw.Count, &kw.AvgWeight, &total); err != nil {
			return nil, err
		}
		keywords = append(keywords, kw)
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return &keywords, nil
}

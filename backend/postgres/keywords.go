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
	s.db.logger.DebugContext(ctx, "Start find keyword summary")
	defer s.db.logger.DebugContext(ctx, "Finish find keyword summary")

	summary, err := getKeywordSummary(ctx, s.db.db, limit)
	if err != nil {
		return nil, err
	}

	return summary, nil
}

func getKeywordSummary(ctx context.Context, db *sql.DB, limit int) (*[]analogdb.KeywordSummary, error) {
	query := `
			SELECT
				word,
				count(word) as count,
				COUNT(*) OVER() as total
			FROM keywords
			GROUP BY word
			ORDER BY count DESC
			LIMIT $1
	`

	arg := limit

	rows, err := db.QueryContext(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keywords := make([]analogdb.KeywordSummary, 0)
	var kw analogdb.KeywordSummary
	var total int
	for rows.Next() {
		if err := rows.Scan(&kw.Word, &kw.Count, &total); err != nil {
			return nil, err
		}
		keywords = append(keywords, kw)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &keywords, nil
}

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

func (s *KeywordService) GetKeywordSummary(ctx context.Context, filter *analogdb.KeywordFilter) (*[]analogdb.KeywordSummary, error) {
	s.db.logger.DebugContext(ctx, "Start find keyword summary")
	defer s.db.logger.DebugContext(ctx, "Finish find keyword summary")

	summary, err := getKeywordSummary(ctx, s.db.db, filter)
	if err != nil {
		return nil, err
	}

	return summary, nil
}

func (s *KeywordService) TagCounts(ctx context.Context) (map[string]int, int, error) {
	s.db.logger.DebugContext(ctx, "Start find tag counts")
	defer s.db.logger.DebugContext(ctx, "Finish find tag counts")

	return getTagCounts(ctx, s.db.db)
}

func getKeywordSummary(ctx context.Context, db *sql.DB, filter *analogdb.KeywordFilter) (*[]analogdb.KeywordSummary, error) {
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

	var limit *int
	if filter != nil {
		limit = filter.Limit
	}
	args := []any{limit}

	if filter != nil && filter.Days != nil {
		query = `
			SELECT
				k.word,
				count(k.word) as count,
				COUNT(*) OVER() as total
			FROM keywords k
			JOIN pictures p ON p.id = k.post_id
			WHERE p.created >= now() - make_interval(days => $2)
			GROUP BY k.word
			ORDER BY count DESC
			LIMIT $1
	`
		args = append(args, *filter.Days)
	}

	rows, err := db.QueryContext(ctx, query, args...)
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

func getTagCounts(ctx context.Context, db *sql.DB) (map[string]int, int, error) {
	rows, err := db.QueryContext(ctx, `SELECT word, count(DISTINCT post_id) FROM keywords GROUP BY word`)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var word string
		var count int
		if err := rows.Scan(&word, &count); err != nil {
			return nil, 0, err
		}
		counts[word] = count
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int
	if err := db.QueryRowContext(ctx, `SELECT count(DISTINCT post_id) FROM keywords`).Scan(&total); err != nil {
		return nil, 0, err
	}
	return counts, total, nil
}

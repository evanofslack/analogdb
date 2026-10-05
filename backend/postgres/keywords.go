package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/evanofslack/analogdb"
	"github.com/lib/pq"
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

	if filter != nil && filter.TopPosts != nil && len(*summary) > 0 {
		if err := s.db.attachKeywordTopPosts(ctx, *summary, *filter.TopPosts); err != nil {
			return nil, err
		}
	}

	return summary, nil
}

func (s *KeywordService) FindKeyword(ctx context.Context, word string, topPosts int) (*analogdb.KeywordDetail, error) {
	s.db.logger.DebugContext(ctx, "Start find keyword", "word", word)
	defer s.db.logger.DebugContext(ctx, "Finish find keyword", "word", word)

	var count int
	if err := s.db.db.QueryRowContext(ctx, `SELECT count(DISTINCT post_id) FROM keywords WHERE word = $1`, word).Scan(&count); err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "keyword not found"}
	}

	top, err := s.db.findKeywordTopPosts(ctx, []string{word}, topPosts)
	if err != nil {
		return nil, err
	}
	return &analogdb.KeywordDetail{Word: word, Count: count, TopPosts: top[catalogKey(word, "")]}, nil
}

func (s *KeywordService) CoOccurring(ctx context.Context, word string) (map[string]int, error) {
	s.db.logger.DebugContext(ctx, "Start find co-occurring keywords", "word", word)
	defer s.db.logger.DebugContext(ctx, "Finish find co-occurring keywords", "word", word)

	query := `
		SELECT k2.word, count(*)
		FROM keywords k1
		JOIN keywords k2 ON k2.post_id = k1.post_id AND k2.word <> k1.word
		WHERE k1.word = $1
		GROUP BY k2.word`

	rows, err := s.db.db.QueryContext(ctx, query, word)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var other string
		var count int
		if err := rows.Scan(&other, &count); err != nil {
			return nil, err
		}
		counts[other] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return counts, nil
}

// attachKeywordTopPosts adds the highest scoring non nsfw posts to each keyword
func (db *DB) attachKeywordTopPosts(ctx context.Context, keywords []analogdb.KeywordSummary, n int) error {
	words := make([]string, 0, len(keywords))
	for _, kw := range keywords {
		words = append(words, kw.Word)
	}
	top, err := db.findKeywordTopPosts(ctx, words, n)
	if err != nil {
		return err
	}
	for i := range keywords {
		keywords[i].TopPosts = top[catalogKey(keywords[i].Word, "")]
	}
	return nil
}

// findKeywordTopPosts finds the top n non nsfw posts of each word, from all time
func (db *DB) findKeywordTopPosts(ctx context.Context, words []string, n int) (map[string][]analogdb.CatalogPost, error) {
	query := `
		SELECT
			word, '' AS name, id,
			COALESCE(title, ''), COALESCE(score, 0),
			COALESCE(lowurl, ''), COALESCE(lowwidth, 0), COALESCE(lowheight, 0),
			COALESCE(medurl, ''), COALESCE(medwidth, 0), COALESCE(medheight, 0)
		FROM (
			SELECT
				k.word, p.id, p.title, p.score, p.lowurl, p.lowwidth, p.lowheight, p.medurl, p.medwidth, p.medheight,
				row_number() OVER (PARTITION BY k.word ORDER BY p.score DESC NULLS LAST, p.id DESC) AS rn
			FROM keywords k
			JOIN pictures p ON p.id = k.post_id
			WHERE p.nsfw = false
			AND k.word = ANY($1)
		) ranked
		WHERE rn <= $2
		ORDER BY word, rn`

	top, err := db.findCatalogTopPosts(ctx, query, pq.Array(words), n)
	if err != nil {
		db.logger.ErrorContext(ctx, "Find keyword top posts", "error", err)
		return nil, err
	}
	return top, nil
}

func (s *KeywordService) TagCounts(ctx context.Context) (map[string]int, int, error) {
	s.db.logger.DebugContext(ctx, "Start find tag counts")
	defer s.db.logger.DebugContext(ctx, "Finish find tag counts")

	return getTagCounts(ctx, s.db.db)
}

func getKeywordSummary(ctx context.Context, db *sql.DB, filter *analogdb.KeywordFilter) (*[]analogdb.KeywordSummary, error) {
	var limit *int
	if filter != nil {
		limit = filter.Limit
	}
	args := []any{limit}

	join, where, having := "", "", ""
	if filter != nil && filter.Days != nil {
		args = append(args, *filter.Days)
		join = "JOIN pictures p ON p.id = k.post_id"
		where = fmt.Sprintf("WHERE p.created >= now() - make_interval(days => $%d)", len(args))
	}
	if filter != nil && filter.MinCount != nil {
		args = append(args, *filter.MinCount)
		having = fmt.Sprintf("HAVING count(k.word) >= $%d", len(args))
	}

	query := fmt.Sprintf(`
			SELECT
				k.word,
				count(k.word) as count,
				COUNT(*) OVER() as total
			FROM keywords k
			%s
			%s
			GROUP BY k.word
			%s
			ORDER BY count DESC
			LIMIT $1
	`, join, where, having)

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

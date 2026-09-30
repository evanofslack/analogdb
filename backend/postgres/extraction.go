package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/evanofslack/analogdb"
	"github.com/lib/pq"
)

// ensure interface is implemented
var _ analogdb.ExtractionService = (*ExtractionService)(nil)

type ExtractionService struct {
	db *DB
}

func NewExtractionService(db *DB) *ExtractionService {
	return &ExtractionService{db: db}
}

func (s *ExtractionService) UpsertExtractions(ctx context.Context, extractions []*analogdb.PostExtraction) (int, []int, error) {
	s.db.logger.DebugContext(ctx, "Starting upsert extractions", "count", len(extractions))
	defer s.db.logger.DebugContext(ctx, "Finished upsert extractions")

	skipped := []int{}
	if len(extractions) == 0 {
		return 0, skipped, nil
	}

	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, nil, err
	}
	defer tx.Rollback()

	ids := make([]int64, 0, len(extractions))
	for _, e := range extractions {
		ids = append(ids, int64(e.PostID))
	}
	rows, err := tx.QueryContext(ctx, `SELECT id FROM pictures WHERE id = ANY($1)`, pq.Array(ids))
	if err != nil {
		return 0, nil, fmt.Errorf("find posts: %w", err)
	}
	exists := make(map[int]bool, len(ids))
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, nil, err
		}
		exists[id] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO post_extractions
			(post_id, extractor_version, model, input, input_hash, raw, unmatched)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (post_id) DO UPDATE SET
			extractor_version = EXCLUDED.extractor_version,
			model = EXCLUDED.model,
			input = EXCLUDED.input,
			input_hash = EXCLUDED.input_hash,
			raw = EXCLUDED.raw,
			unmatched = EXCLUDED.unmatched,
			updated = NOW()`)
	if err != nil {
		return 0, nil, err
	}
	defer stmt.Close()

	written := 0
	for _, e := range extractions {
		if !exists[e.PostID] {
			skipped = append(skipped, e.PostID)
			continue
		}
		unmatched := e.Unmatched
		if len(unmatched) == 0 {
			unmatched = json.RawMessage(`[]`)
		}
		if _, err := stmt.ExecContext(ctx, e.PostID, e.ExtractorVersion, e.Model, e.Input,
			e.InputHash, []byte(e.Raw), []byte(unmatched)); err != nil {
			return 0, nil, fmt.Errorf("upsert extraction for post %d: %w", e.PostID, err)
		}
		written++
	}

	if err := tx.Commit(); err != nil {
		return 0, nil, err
	}
	return written, skipped, nil
}

func (s *ExtractionService) FindExtractions(ctx context.Context, filter *analogdb.ExtractionFilter) ([]*analogdb.PostExtraction, error) {
	s.db.logger.DebugContext(ctx, "Starting find extractions")
	defer s.db.logger.DebugContext(ctx, "Finished find extractions")

	where, args := []string{}, []any{}
	add := func(clause string, arg any) {
		args = append(args, arg)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}
	if filter.HasUnmatched != nil {
		add("has_unmatched = $%d", *filter.HasUnmatched)
	}
	if filter.Kind != nil {
		contains, _ := json.Marshal([]map[string]string{{"kind": *filter.Kind}})
		add("unmatched @> $%d::jsonb", string(contains))
	}
	if filter.Key != nil {
		contains, _ := json.Marshal([]map[string]string{{"key": *filter.Key}})
		add("unmatched @> $%d::jsonb", string(contains))
	}
	if filter.BeforeID != nil {
		add("post_id < $%d", *filter.BeforeID)
	}

	full := "'', NULL::jsonb"
	if filter.Full {
		full = "input, raw"
	}
	query := `SELECT post_id, extractor_version, model, input_hash, unmatched, created, updated, ` + full + `
		FROM post_extractions`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	args = append(args, filter.Limit)
	query += fmt.Sprintf(" ORDER BY post_id DESC LIMIT $%d", len(args))

	rows, err := s.db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	extractions := make([]*analogdb.PostExtraction, 0)
	for rows.Next() {
		var e analogdb.PostExtraction
		var unmatched, raw []byte
		if err := rows.Scan(&e.PostID, &e.ExtractorVersion, &e.Model, &e.InputHash, &unmatched,
			&e.Created, &e.Updated, &e.Input, &raw); err != nil {
			return nil, err
		}
		e.Unmatched = json.RawMessage(unmatched)
		if raw != nil {
			e.Raw = json.RawMessage(raw)
		}
		extractions = append(extractions, &e)
	}
	return extractions, rows.Err()
}

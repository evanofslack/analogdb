package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/evanofslack/analogdb"
	"github.com/lib/pq"
)

// ensure interface is implemented
var _ analogdb.StatsService = (*StatsService)(nil)

type StatsService struct {
	db *DB
}

func NewStatsService(db *DB) *StatsService {
	return &StatsService{db: db}
}

// ReplaceSeed swaps the whole seed in one transaction, so themes from an
// older fit never mix with a newer one
func (s *StatsService) ReplaceSeed(ctx context.Context, seed *analogdb.StatsSeed) error {
	s.db.logger.DebugContext(ctx, "Starting replace stats seed", "version", seed.Version)
	defer s.db.logger.DebugContext(ctx, "Finished replace stats seed")

	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM stats_themes`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM stats_prompts`); err != nil {
		return err
	}
	for _, t := range seed.Themes {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO stats_themes (key, name, hidden, role, center) VALUES ($1, $2, $3, $4, $5)`,
			t.Key, t.Name, t.Hidden, t.Role, pq.Float32Array(t.Center))
		if err != nil {
			return err
		}
	}
	for _, p := range seed.Prompts {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO stats_prompts (key, label, text, vector) VALUES ($1, $2, $3, $4)`,
			p.Key, p.Label, p.Text, pq.Float32Array(p.Vector))
		if err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO stats_seed (id, version, model) VALUES (1, $1, $2)
		ON CONFLICT (id) DO UPDATE SET version = EXCLUDED.version, model = EXCLUDED.model, updated = NOW()`,
		seed.Version, seed.Model)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		s.db.logger.ErrorContext(ctx, "Fail replace stats seed", "error", err)
		return err
	}
	s.db.logger.InfoContext(ctx, "Replaced stats seed", "version", seed.Version, "themes", len(seed.Themes), "prompts", len(seed.Prompts))
	return nil
}

func (s *StatsService) FindSeed(ctx context.Context) (*analogdb.StatsSeed, error) {
	s.db.logger.DebugContext(ctx, "Starting find stats seed")
	defer s.db.logger.DebugContext(ctx, "Finished find stats seed")

	seed := &analogdb.StatsSeed{Themes: []analogdb.StatsTheme{}, Prompts: []analogdb.StatsPrompt{}}
	err := s.db.db.QueryRowContext(ctx, `SELECT version, model FROM stats_seed WHERE id = 1`).Scan(&seed.Version, &seed.Model)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "stats seed not found"}
	}
	if err != nil {
		return nil, err
	}

	rows, err := s.db.db.QueryContext(ctx, `SELECT key, name, hidden, role, center FROM stats_themes ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var t analogdb.StatsTheme
		var center pq.Float32Array
		if err := rows.Scan(&t.Key, &t.Name, &t.Hidden, &t.Role, &center); err != nil {
			return nil, err
		}
		t.Center = center
		seed.Themes = append(seed.Themes, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = s.db.db.QueryContext(ctx, `SELECT key, label, text, vector FROM stats_prompts ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p analogdb.StatsPrompt
		var vector pq.Float32Array
		if err := rows.Scan(&p.Key, &p.Label, &p.Text, &vector); err != nil {
			return nil, err
		}
		p.Vector = vector
		seed.Prompts = append(seed.Prompts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return seed, nil
}

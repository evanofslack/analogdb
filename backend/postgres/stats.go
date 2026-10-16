package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

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

const statsPostsQuery = `
	SELECT p.id, COALESCE(p.author, ''), p.time, COALESCE(p.score, 0),
		COALESCE(p.nsfw, false), COALESCE(p.greyscale, false),
		COALESCE(f.film_make, ''), COALESCE(f.film_type, ''), COALESCE(f.color_type, ''),
		COALESCE(c.camera_make, ''), COALESCE(c.camera_model, ''),
		COALESCE(p.focal_length, 0), COALESCE(p.aperture, ''), COALESCE(p.title, ''),
		COALESCE(p.lowurl, ''), COALESCE(p.lowwidth, 0), COALESCE(p.lowheight, 0),
		COALESCE(p.medurl, ''), COALESCE(p.medwidth, 0), COALESCE(p.medheight, 0),
		percent_rank() OVER (
			PARTITION BY date_trunc('month', to_timestamp(p.time) AT TIME ZONE 'UTC')
			ORDER BY COALESCE(p.score, 0)
		)
	FROM pictures p
	LEFT JOIN films f ON f.film_make = p.film_make AND f.film_type = p.film_type
	LEFT JOIN cameras c ON c.camera_make = p.camera_make AND c.camera_model = p.camera_model
	WHERE p.time >= $1
	ORDER BY p.id`

const statsColorsQuery = `
	SELECT c.post_id, COALESCE(c.percent, 0), c.oklab_l, c.oklab_a, c.oklab_b, c.chroma, c.hue
	FROM colors c
	JOIN pictures p ON p.id = c.post_id
	WHERE p.time >= $1 AND c.oklab_l IS NOT NULL AND c.chroma IS NOT NULL AND c.hue IS NOT NULL
	ORDER BY c.post_id, c.id`

const statsKeywordsQuery = `
	SELECT DISTINCT k.post_id, k.word
	FROM keywords k
	JOIN pictures p ON p.id = k.post_id
	WHERE p.time >= $1 AND k.word = ANY($2)
	ORDER BY k.post_id, k.word`

// LoadStatsInput reads every post since the window start with its colors and
// the keywords among subjects, for the stats builder
func (s *StatsService) LoadStatsInput(ctx context.Context, since time.Time, subjects []string) (*analogdb.StatsInput, error) {
	s.db.logger.DebugContext(ctx, "Starting load stats input")
	defer s.db.logger.DebugContext(ctx, "Finished load stats input")

	start := since.Unix()
	in := &analogdb.StatsInput{}

	rows, err := s.db.db.QueryContext(ctx, statsPostsQuery, start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p analogdb.StatsInputPost
		var lowURL, medURL string
		var lowWidth, lowHeight, medWidth, medHeight int
		err := rows.Scan(&p.ID, &p.Author, &p.Time, &p.Score, &p.Nsfw, &p.Greyscale,
			&p.FilmMake, &p.FilmType, &p.FilmColorType, &p.CameraMake, &p.CameraModel,
			&p.FocalLength, &p.Aperture, &p.Title,
			&lowURL, &lowWidth, &lowHeight, &medURL, &medWidth, &medHeight, &p.Popularity)
		if err != nil {
			return nil, err
		}
		if p.FilmMake != "" {
			p.FilmSlug = analogdb.Slug(p.FilmMake, p.FilmType)
		}
		if p.CameraMake != "" {
			p.CameraSlug = analogdb.Slug(p.CameraMake, p.CameraModel)
		}
		p.Images = []analogdb.Image{
			{Label: "low", Url: imageURL(s.db.imageBase, lowURL), Width: lowWidth, Height: lowHeight},
			{Label: "medium", Url: imageURL(s.db.imageBase, medURL), Width: medWidth, Height: medHeight},
		}
		in.Posts = append(in.Posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = s.db.db.QueryContext(ctx, statsColorsQuery, start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var c analogdb.StatsInputColor
		if err := rows.Scan(&c.PostID, &c.Percent, &c.L, &c.A, &c.B, &c.Chroma, &c.Hue); err != nil {
			return nil, err
		}
		in.Colors = append(in.Colors, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = s.db.db.QueryContext(ctx, statsKeywordsQuery, start, pq.Array(subjects))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var k analogdb.StatsInputKeyword
		if err := rows.Scan(&k.PostID, &k.Word); err != nil {
			return nil, err
		}
		in.Keywords = append(in.Keywords, k)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return in, nil
}

const statsSnapshotsKept = 10

// CreateSnapshot stores a build and drops all but the newest ten
func (s *StatsService) CreateSnapshot(ctx context.Context, snapshot *analogdb.StatsSnapshot) error {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx, `
		INSERT INTO stats_snapshots (seed_version, build_ms, body) VALUES ($1, $2, $3)
		RETURNING id, created`,
		snapshot.SeedVersion, snapshot.BuildMs, []byte(snapshot.Body)).Scan(&snapshot.ID, &snapshot.Created)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		DELETE FROM stats_snapshots WHERE id NOT IN (
			SELECT id FROM stats_snapshots ORDER BY id DESC LIMIT $1
		)`, statsSnapshotsKept)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	snapshot.Size = len(snapshot.Body)
	s.db.logger.InfoContext(ctx, "Created stats snapshot", "id", snapshot.ID, "build_ms", snapshot.BuildMs, "size", snapshot.Size)
	return nil
}

func (s *StatsService) FindLatestSnapshot(ctx context.Context) (*analogdb.StatsSnapshot, error) {
	var snapshot analogdb.StatsSnapshot
	var body []byte
	err := s.db.db.QueryRowContext(ctx, `
		SELECT id, created, seed_version, build_ms, body FROM stats_snapshots ORDER BY id DESC LIMIT 1`,
	).Scan(&snapshot.ID, &snapshot.Created, &snapshot.SeedVersion, &snapshot.BuildMs, &body)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "stats not built yet"}
	}
	if err != nil {
		return nil, err
	}
	snapshot.Body = body
	snapshot.Size = len(body)
	return &snapshot, nil
}

// FindSnapshots lists the stored builds newest first, without their bodies
func (s *StatsService) FindSnapshots(ctx context.Context) ([]*analogdb.StatsSnapshot, error) {
	rows, err := s.db.db.QueryContext(ctx, `
		SELECT id, created, seed_version, build_ms, octet_length(body::text) FROM stats_snapshots ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	snapshots := []*analogdb.StatsSnapshot{}
	for rows.Next() {
		var snapshot analogdb.StatsSnapshot
		if err := rows.Scan(&snapshot.ID, &snapshot.Created, &snapshot.SeedVersion, &snapshot.BuildMs, &snapshot.Size); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, &snapshot)
	}
	return snapshots, rows.Err()
}

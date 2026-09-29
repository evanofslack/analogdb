package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/evanofslack/analogdb"
)

// ensure interface is implemented
var _ analogdb.AdminService = (*AdminService)(nil)

var adminTables = []string{"pictures", "keywords", "colors", "post_updates", "cameras", "films"}

type AdminService struct {
	db *DB
}

func NewAdminService(db *DB) *AdminService {
	return &AdminService{db: db}
}

func (s *AdminService) Stats(ctx context.Context) (*analogdb.AdminStats, error) {
	s.db.logger.DebugContext(ctx, "Starting admin stats")
	defer s.db.logger.DebugContext(ctx, "Finished admin stats")

	stats := &analogdb.AdminStats{}
	if err := adminCounts(ctx, s.db.db, stats); err != nil {
		return nil, fmt.Errorf("admin counts: %w", err)
	}
	if err := adminFreshness(ctx, s.db.db, &stats.Freshness); err != nil {
		return nil, fmt.Errorf("admin freshness: %w", err)
	}
	if err := adminDatabase(ctx, s.db.db, &stats.Database); err != nil {
		return nil, fmt.Errorf("admin database: %w", err)
	}
	return stats, nil
}

func adminCounts(ctx context.Context, db *sql.DB, stats *analogdb.AdminStats) error {
	query := `
		SELECT
			(SELECT COUNT(*) FROM pictures),
			(SELECT COUNT(DISTINCT author) FROM pictures),
			(SELECT COUNT(*) FROM cameras),
			(SELECT COUNT(*) FROM films),
			(SELECT COUNT(DISTINCT word) FROM keywords),
			COUNT(*) FILTER (WHERE time >= EXTRACT(EPOCH FROM NOW())::bigint - 86400),
			COUNT(*) FILTER (WHERE time >= EXTRACT(EPOCH FROM NOW())::bigint - 7 * 86400),
			COUNT(*) FILTER (WHERE time >= EXTRACT(EPOCH FROM NOW())::bigint - 30 * 86400)
		FROM pictures
		WHERE time >= EXTRACT(EPOCH FROM NOW())::bigint - 30 * 86400`
	c := &stats.Counts
	p := &stats.Posted
	return db.QueryRowContext(ctx, query).Scan(
		&c.Posts, &c.Authors, &c.Cameras, &c.Films, &c.Keywords,
		&p.Day, &p.Week, &p.Month,
	)
}

func adminFreshness(ctx context.Context, db *sql.DB, f *analogdb.AdminFreshness) error {
	query := `
		SELECT
			(SELECT MAX(time) FROM pictures),
			MAX(score_update_time),
			MAX(keywords_update_time),
			MAX(colors_update_time)
		FROM post_updates`
	var newest, score, keywords, colors sql.NullInt64
	if err := db.QueryRowContext(ctx, query).Scan(&newest, &score, &keywords, &colors); err != nil {
		return err
	}
	f.NewestPost = nullInt64Ptr(newest)
	f.ScoreUpdate = nullInt64Ptr(score)
	f.KeywordsUpdate = nullInt64Ptr(keywords)
	f.ColorsUpdate = nullInt64Ptr(colors)
	return nil
}

func adminDatabase(ctx context.Context, db *sql.DB, d *analogdb.AdminDatabase) error {
	if err := db.QueryRowContext(ctx, `SELECT pg_database_size(current_database())`).Scan(&d.Bytes); err != nil {
		return err
	}

	d.Tables = make([]analogdb.AdminTableSize, 0, len(adminTables))
	for _, table := range adminTables {
		size := analogdb.AdminTableSize{Name: table}
		if err := db.QueryRowContext(ctx, `SELECT pg_total_relation_size($1::regclass)`, table).Scan(&size.Bytes); err != nil {
			return err
		}
		d.Tables = append(d.Tables, size)
	}

	err := db.QueryRowContext(ctx, `SELECT version, dirty FROM schema_migrations LIMIT 1`).Scan(&d.MigrationVersion, &d.MigrationDirty)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	return nil
}

func (s *AdminService) Quality(ctx context.Context) (*analogdb.AdminQuality, error) {
	s.db.logger.DebugContext(ctx, "Starting admin quality")
	defer s.db.logger.DebugContext(ctx, "Finished admin quality")

	quality := &analogdb.AdminQuality{}
	if err := adminCoverage(ctx, s.db.db, &quality.Coverage); err != nil {
		return nil, fmt.Errorf("admin coverage: %w", err)
	}

	cameras, err := unmatchedCameras(ctx, s.db.db)
	if err != nil {
		return nil, fmt.Errorf("unmatched cameras: %w", err)
	}
	quality.UnmatchedCameras = cameras

	films, err := unmatchedFilms(ctx, s.db.db)
	if err != nil {
		return nil, fmt.Errorf("unmatched films: %w", err)
	}
	quality.UnmatchedFilms = films

	return quality, nil
}

func adminCoverage(ctx context.Context, db *sql.DB, c *analogdb.AdminCoverage) error {
	query := `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE NULLIF(p.camera_make, '') IS NOT NULL),
			COUNT(*) FILTER (WHERE NULLIF(p.film_make, '') IS NOT NULL OR NULLIF(p.film_type, '') IS NOT NULL),
			COUNT(*) FILTER (WHERE NULLIF(p.description, '') IS NOT NULL),
			COUNT(*) FILTER (WHERE EXISTS (SELECT 1 FROM keywords k WHERE k.post_id = p.id)),
			COUNT(*) FILTER (WHERE EXISTS (SELECT 1 FROM colors c WHERE c.post_id = p.id)),
			COUNT(*) FILTER (WHERE p.focal_length IS NOT NULL AND p.focal_length > 0),
			COUNT(*) FILTER (WHERE NULLIF(p.aperture, '') IS NOT NULL)
		FROM pictures p`
	return db.QueryRowContext(ctx, query).Scan(
		&c.Total, &c.Camera, &c.Film, &c.Description, &c.Keywords, &c.Colors, &c.FocalLength, &c.Aperture,
	)
}

const unmatchedLimit = 50

func unmatchedCameras(ctx context.Context, db *sql.DB) ([]analogdb.AdminUnmatchedCamera, error) {
	query := `
		SELECT p.camera_make, p.camera_model, COUNT(*) AS post_count, MAX(p.id) AS sample_post_id
		FROM pictures p
		WHERE NULLIF(p.camera_make, '') IS NOT NULL
			AND NULLIF(p.camera_model, '') IS NOT NULL
			AND NOT EXISTS (
				SELECT 1 FROM cameras c
				WHERE c.camera_make = p.camera_make AND c.camera_model = p.camera_model
			)
		GROUP BY p.camera_make, p.camera_model
		ORDER BY post_count DESC, p.camera_make, p.camera_model
		LIMIT $1`
	rows, err := db.QueryContext(ctx, query, unmatchedLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cameras := make([]analogdb.AdminUnmatchedCamera, 0)
	for rows.Next() {
		var c analogdb.AdminUnmatchedCamera
		if err := rows.Scan(&c.CameraMake, &c.CameraModel, &c.PostCount, &c.SamplePostID); err != nil {
			return nil, err
		}
		cameras = append(cameras, c)
	}
	return cameras, rows.Err()
}

// films match the catalog on make and type, the same as the film post counts
func unmatchedFilms(ctx context.Context, db *sql.DB) ([]analogdb.AdminUnmatchedFilm, error) {
	query := `
		SELECT p.film_make, p.film_type, MODE() WITHIN GROUP (ORDER BY p.film_speed) AS film_speed,
			COUNT(*) AS post_count, MAX(p.id) AS sample_post_id
		FROM pictures p
		WHERE NULLIF(p.film_make, '') IS NOT NULL
			AND NULLIF(p.film_type, '') IS NOT NULL
			AND NOT EXISTS (
				SELECT 1 FROM films f
				WHERE f.film_make = p.film_make AND f.film_type = p.film_type
			)
		GROUP BY p.film_make, p.film_type
		ORDER BY post_count DESC, p.film_make, p.film_type
		LIMIT $1`
	rows, err := db.QueryContext(ctx, query, unmatchedLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	films := make([]analogdb.AdminUnmatchedFilm, 0)
	for rows.Next() {
		var f analogdb.AdminUnmatchedFilm
		var speed sql.NullInt64
		if err := rows.Scan(&f.FilmMake, &f.FilmType, &speed, &f.PostCount, &f.SamplePostID); err != nil {
			return nil, err
		}
		f.FilmSpeed = nullInt64Ptr(speed)
		films = append(films, f)
	}
	return films, rows.Err()
}

var missingWhere = map[analogdb.MissingField]string{
	analogdb.MissingCamera:      `NULLIF(p.camera_make, '') IS NULL`,
	analogdb.MissingFilm:        `NULLIF(p.film_make, '') IS NULL AND NULLIF(p.film_type, '') IS NULL`,
	analogdb.MissingDescription: `NULLIF(p.description, '') IS NULL`,
	analogdb.MissingKeywords:    `NOT EXISTS (SELECT 1 FROM keywords k WHERE k.post_id = p.id)`,
	analogdb.MissingColors:      `NOT EXISTS (SELECT 1 FROM colors c WHERE c.post_id = p.id)`,
}

func (s *AdminService) MissingPosts(ctx context.Context, filter *analogdb.MissingPostsFilter) ([]*analogdb.AdminPost, error) {
	s.db.logger.DebugContext(ctx, "Starting admin missing posts", "field", filter.Field)
	defer s.db.logger.DebugContext(ctx, "Finished admin missing posts")

	where, ok := missingWhere[filter.Field]
	if !ok {
		return nil, &analogdb.Error{Code: analogdb.ERRBADREQUEST, Message: fmt.Sprintf("invalid field: %s", filter.Field)}
	}

	args := []any{filter.Limit}
	if filter.BeforeID != nil {
		where += " AND p.id < $2"
		args = append(args, *filter.BeforeID)
	}

	query := `
		SELECT p.id, p.title, p.author, p.time, p.lowurl,
			p.camera_make, p.camera_model, p.film_make, p.film_type, p.film_speed
		FROM pictures p
		WHERE ` + where + `
		ORDER BY p.id DESC
		LIMIT $1`
	rows, err := s.db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := make([]*analogdb.AdminPost, 0)
	for rows.Next() {
		var p analogdb.AdminPost
		var title, author, lowURL, cameraMake, cameraModel, filmMake, filmType sql.NullString
		var postTime, filmSpeed sql.NullInt64
		if err := rows.Scan(&p.ID, &title, &author, &postTime, &lowURL,
			&cameraMake, &cameraModel, &filmMake, &filmType, &filmSpeed); err != nil {
			return nil, err
		}
		p.Title = title.String
		p.Author = author.String
		p.Time = postTime.Int64
		p.LowURL = lowURL.String
		p.CameraMake = nullStringPtr(cameraMake)
		p.CameraModel = nullStringPtr(cameraModel)
		p.FilmMake = nullStringPtr(filmMake)
		p.FilmType = nullStringPtr(filmType)
		p.FilmSpeed = nullInt64Ptr(filmSpeed)
		posts = append(posts, &p)
	}
	return posts, rows.Err()
}

func nullInt64Ptr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	return &n.Int64
}

func nullStringPtr(n sql.NullString) *string {
	if !n.Valid || n.String == "" {
		return nil
	}
	return &n.String
}

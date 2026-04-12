package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/evanofslack/analogdb"
)

type StatsService struct {
	db *DB
}

func NewStatsService(db *DB) *StatsService {
	return &StatsService{db: db}
}

func (s *StatsService) GetOverview(ctx context.Context, filter *analogdb.StatsFilter) (*analogdb.StatsOverview, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	return s.db.getStatsOverview(ctx, tx, filter)
}

func (s *StatsService) GetPostsOverTime(ctx context.Context, filter *analogdb.StatsFilter) ([]*analogdb.StatsPeriod, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	return s.db.getStatsPostsOverTime(ctx, tx, filter)
}

func (s *StatsService) GetFilmStats(ctx context.Context, filter *analogdb.StatsFilter) ([]*analogdb.StatsFilm, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	return s.db.getStatsFilms(ctx, tx, filter)
}

func (s *StatsService) GetCameraStats(ctx context.Context, filter *analogdb.StatsFilter) ([]*analogdb.StatsCamera, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	return s.db.getStatsCameras(ctx, tx, filter)
}

func (s *StatsService) GetColorStats(ctx context.Context, filter *analogdb.StatsFilter) ([]*analogdb.StatsColor, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	return s.db.getStatsColors(ctx, tx, filter)
}

func (s *StatsService) GetKeywordStats(ctx context.Context, filter *analogdb.StatsFilter) ([]*analogdb.StatsKeyword, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	return s.db.getStatsKeywords(ctx, tx, filter)
}

func (db *DB) getStatsOverview(ctx context.Context, tx *sql.Tx, filter *analogdb.StatsFilter) (*analogdb.StatsOverview, error) {
	startArg, endArg := statsTimeArgs(filter)

	query := `
		SELECT
			COUNT(id)                                                           AS total_posts,
			COUNT(DISTINCT author)                                              AS total_authors,
			(SELECT COUNT(DISTINCT word) FROM keywords)                         AS total_keywords,
			(SELECT COUNT(DISTINCT film_make)
			 FROM pictures WHERE film_make IS NOT NULL)                         AS total_film_brands,
			(SELECT COUNT(DISTINCT film_make || film_type)
			 FROM pictures WHERE film_make IS NOT NULL)                         AS total_film_stocks,
			(SELECT COUNT(DISTINCT camera_make)
			 FROM pictures WHERE camera_make IS NOT NULL)                       AS total_camera_brands,
			(SELECT COUNT(DISTINCT camera_make || camera_model)
			 FROM pictures WHERE camera_make IS NOT NULL)                       AS total_camera_models,
			ROUND(AVG(score)::numeric, 2)                                       AS avg_score,
			ROUND(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY score)::numeric, 2) AS median_score,
			MIN(score)                                                          AS min_score,
			MAX(score)                                                          AS max_score,
			ROUND(STDDEV(score)::numeric, 2)                                    AS std_dev_score
		FROM pictures
		WHERE ($1::bigint IS NULL OR time >= $1)
		  AND ($2::bigint IS NULL OR time <= $2)
	`

	var overview analogdb.StatsOverview
	err := tx.QueryRowContext(ctx, query, startArg, endArg).Scan(
		&overview.TotalPosts,
		&overview.TotalAuthors,
		&overview.TotalKeywords,
		&overview.TotalFilmBrands,
		&overview.TotalFilmStocks,
		&overview.TotalCameraBrands,
		&overview.TotalCameraModels,
		&overview.AvgScore,
		&overview.MedianScore,
		&overview.MinScore,
		&overview.MaxScore,
		&overview.StdDevScore,
	)
	if err != nil {
		db.logger.ErrorContext(ctx, "Get stats overview", "error", err)
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &overview, nil
}

func (db *DB) getStatsPostsOverTime(ctx context.Context, tx *sql.Tx, filter *analogdb.StatsFilter) ([]*analogdb.StatsPeriod, error) {
	granularity := "month"
	if filter != nil && filter.Granularity != nil {
		granularity = *filter.Granularity
	}

	startArg, endArg := statsTimeArgs(filter)

	query := fmt.Sprintf(`
		SELECT
			TO_CHAR(DATE_TRUNC('%s', TO_TIMESTAMP(time)), 'YYYY-MM-DD') AS period,
			COUNT(id)                                                     AS count,
			ROUND(AVG(score)::numeric, 2)                                AS avg_score
		FROM pictures
		WHERE ($1::bigint IS NULL OR time >= $1)
		  AND ($2::bigint IS NULL OR time <= $2)
		GROUP BY DATE_TRUNC('%s', TO_TIMESTAMP(time))
		ORDER BY DATE_TRUNC('%s', TO_TIMESTAMP(time)) ASC
	`, granularity, granularity, granularity)

	rows, err := tx.QueryContext(ctx, query, startArg, endArg)
	if err != nil {
		db.logger.ErrorContext(ctx, "Get stats posts over time", "error", err)
		return nil, err
	}
	defer rows.Close()

	var periods []*analogdb.StatsPeriod
	for rows.Next() {
		var p analogdb.StatsPeriod
		if err := rows.Scan(&p.Period, &p.Count, &p.AvgScore); err != nil {
			db.logger.ErrorContext(ctx, "Get stats posts over time, scan error", "error", err)
			return nil, err
		}
		periods = append(periods, &p)
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return periods, nil
}

func (db *DB) getStatsFilms(ctx context.Context, tx *sql.Tx, filter *analogdb.StatsFilter) ([]*analogdb.StatsFilm, error) {
	orderCol := statsOrderCol(filter)
	limit := statsLimit(filter)
	startArg, endArg := statsTimeArgs(filter)

	query := fmt.Sprintf(`
		SELECT
			p.film_make,
			p.film_type,
			COALESCE(f.film_speed, 0)           AS film_speed,
			COALESCE(f.color_type, '')           AS color_type,
			COUNT(p.id)                          AS post_count,
			ROUND(AVG(p.score)::numeric, 2)      AS avg_score
		FROM pictures p
		LEFT JOIN films f ON p.film_make = f.film_make AND p.film_type = f.film_type
		WHERE p.film_make IS NOT NULL
		  AND p.film_type IS NOT NULL
		  AND ($1::bigint IS NULL OR p.time >= $1)
		  AND ($2::bigint IS NULL OR p.time <= $2)
		GROUP BY p.film_make, p.film_type, f.film_speed, f.color_type
		ORDER BY %s DESC
		LIMIT %d
	`, orderCol, limit)

	rows, err := tx.QueryContext(ctx, query, startArg, endArg)
	if err != nil {
		db.logger.ErrorContext(ctx, "Get stats films", "error", err)
		return nil, err
	}
	defer rows.Close()

	var films []*analogdb.StatsFilm
	for rows.Next() {
		var f analogdb.StatsFilm
		if err := rows.Scan(&f.FilmMake, &f.FilmType, &f.FilmSpeed, &f.ColorType, &f.PostCount, &f.AvgScore); err != nil {
			db.logger.ErrorContext(ctx, "Get stats films, scan error", "error", err)
			return nil, err
		}
		films = append(films, &f)
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return films, nil
}

func (db *DB) getStatsCameras(ctx context.Context, tx *sql.Tx, filter *analogdb.StatsFilter) ([]*analogdb.StatsCamera, error) {
	orderCol := statsOrderCol(filter)
	limit := statsLimit(filter)
	startArg, endArg := statsTimeArgs(filter)

	query := fmt.Sprintf(`
		SELECT
			p.camera_make,
			p.camera_model,
			COUNT(p.id)                          AS post_count,
			ROUND(AVG(p.score)::numeric, 2)      AS avg_score
		FROM pictures p
		WHERE p.camera_make IS NOT NULL
		  AND p.camera_model IS NOT NULL
		  AND ($1::bigint IS NULL OR p.time >= $1)
		  AND ($2::bigint IS NULL OR p.time <= $2)
		GROUP BY p.camera_make, p.camera_model
		ORDER BY %s DESC
		LIMIT %d
	`, orderCol, limit)

	rows, err := tx.QueryContext(ctx, query, startArg, endArg)
	if err != nil {
		db.logger.ErrorContext(ctx, "Get stats cameras", "error", err)
		return nil, err
	}
	defer rows.Close()

	var cameras []*analogdb.StatsCamera
	for rows.Next() {
		var c analogdb.StatsCamera
		if err := rows.Scan(&c.CameraMake, &c.CameraModel, &c.PostCount, &c.AvgScore); err != nil {
			db.logger.ErrorContext(ctx, "Get stats cameras, scan error", "error", err)
			return nil, err
		}
		cameras = append(cameras, &c)
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return cameras, nil
}

func (db *DB) getStatsColors(ctx context.Context, tx *sql.Tx, filter *analogdb.StatsFilter) ([]*analogdb.StatsColor, error) {
	orderCol := statsOrderCol(filter)
	limit := statsLimit(filter)
	startArg, endArg := statsTimeArgs(filter)

	query := fmt.Sprintf(`
		SELECT
			c.html                                   AS html_name,
			MIN(c.hex)                               AS hex,
			COUNT(DISTINCT c.post_id)                AS post_count,
			ROUND(AVG(c.percent)::numeric, 3)        AS avg_percent,
			ROUND(AVG(p.score)::numeric, 2)          AS avg_score
		FROM colors c
		JOIN pictures p ON c.post_id = p.id
		WHERE c.html IS NOT NULL
		  AND c.html != ''
		  AND ($1::bigint IS NULL OR p.time >= $1)
		  AND ($2::bigint IS NULL OR p.time <= $2)
		GROUP BY c.html
		ORDER BY %s DESC
		LIMIT %d
	`, orderCol, limit)

	rows, err := tx.QueryContext(ctx, query, startArg, endArg)
	if err != nil {
		db.logger.ErrorContext(ctx, "Get stats colors", "error", err)
		return nil, err
	}
	defer rows.Close()

	var colors []*analogdb.StatsColor
	for rows.Next() {
		var c analogdb.StatsColor
		if err := rows.Scan(&c.HtmlName, &c.Hex, &c.PostCount, &c.AvgPercent, &c.AvgScore); err != nil {
			db.logger.ErrorContext(ctx, "Get stats colors, scan error", "error", err)
			return nil, err
		}
		colors = append(colors, &c)
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return colors, nil
}

func (db *DB) getStatsKeywords(ctx context.Context, tx *sql.Tx, filter *analogdb.StatsFilter) ([]*analogdb.StatsKeyword, error) {
	orderCol := statsOrderCol(filter)
	limit := statsLimit(filter)
	startArg, endArg := statsTimeArgs(filter)

	query := fmt.Sprintf(`
		SELECT
			k.word,
			COUNT(DISTINCT k.post_id)                AS post_count,
			ROUND(AVG(p.score)::numeric, 2)          AS avg_score
		FROM keywords k
		JOIN pictures p ON k.post_id = p.id
		WHERE k.word IS NOT NULL
		  AND k.word != ''
		  AND ($1::bigint IS NULL OR p.time >= $1)
		  AND ($2::bigint IS NULL OR p.time <= $2)
		GROUP BY k.word
		ORDER BY %s DESC
		LIMIT %d
	`, orderCol, limit)

	rows, err := tx.QueryContext(ctx, query, startArg, endArg)
	if err != nil {
		db.logger.ErrorContext(ctx, "Get stats keywords", "error", err)
		return nil, err
	}
	defer rows.Close()

	var keywords []*analogdb.StatsKeyword
	for rows.Next() {
		var k analogdb.StatsKeyword
		if err := rows.Scan(&k.Word, &k.PostCount, &k.AvgScore); err != nil {
			db.logger.ErrorContext(ctx, "Get stats keywords, scan error", "error", err)
			return nil, err
		}
		keywords = append(keywords, &k)
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return keywords, nil
}

func statsTimeArgs(filter *analogdb.StatsFilter) (startArg, endArg interface{}) {
	startArg = nil
	endArg = nil
	if filter == nil {
		return
	}
	if filter.Start != nil {
		startArg = *filter.Start
	}
	if filter.End != nil {
		endArg = *filter.End
	}
	return
}

func statsOrderCol(filter *analogdb.StatsFilter) string {
	if filter != nil && filter.Metric != nil && *filter.Metric == "score" {
		return "avg_score"
	}
	return "post_count"
}

func statsLimit(filter *analogdb.StatsFilter) int {
	const defaultLimit = 20
	if filter != nil && filter.Limit != nil && *filter.Limit > 0 {
		return *filter.Limit
	}
	return defaultLimit
}

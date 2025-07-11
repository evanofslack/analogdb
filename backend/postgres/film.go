package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/evanofslack/analogdb"
)

type FilmService struct {
	db *DB
}

func NewFilmService(db *DB) *FilmService {
	return &FilmService{db: db}
}

func (s *FilmService) AllFilms(ctx context.Context, filter *analogdb.FilmFilter) ([]*analogdb.Film, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	return s.db.findFilms(ctx, tx, filter)
}

func (s *FilmService) CreateFilm(ctx context.Context, film *analogdb.CreateFilm) (*analogdb.CreateFilm, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	created, err := s.db.createFilm(ctx, tx, film)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (db *DB) createFilm(ctx context.Context, tx *sql.Tx, film *analogdb.CreateFilm) (*analogdb.CreateFilm, error) {
	var id int64

	query := `
	INSERT INTO films
        (film_make, film_type, film_speed, color_type, description)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (film_make, film_type, film_speed) 
        DO UPDATE SET 
            color_type = EXCLUDED.color_type,
            description = EXCLUDED.description,
            updated = CURRENT_TIMESTAMP
        RETURNING id
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		db.logger.Error().Ctx(ctx).Err(err).Int64("film_id", id).Msg("Insert film")
		return nil, err
	}
	defer stmt.Close()

	err = stmt.QueryRowContext(
		ctx,
		film.Make,
		film.Type,
		film.Speed,
		film.ColorType,
		film.Description).Scan(&id)
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int64("film_id", id).Msg("Insert film")
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	db.logger.Info().Ctx(ctx).Int64("film_id", id).Msg("Finished inserting film")
	film.Id = int(id)

	return film, nil
}

// findFilms is the general function responsible for handling all film queries.
func (db *DB) findFilms(ctx context.Context, tx *sql.Tx, filter *analogdb.FilmFilter) ([]*analogdb.Film, error) {
	db.logger.Debug().Ctx(ctx).Msg("Starting find films")
	defer db.logger.Debug().Ctx(ctx).Msg("Finished find films")

	query := `
    SELECT 
    id,
    film_make,
    film_type,
    film_speed,
    color_type,
    description,
    created,
    updated,
    0 as post_count,
    0 as total_post_count
FROM films
ORDER BY film_make, film_type, film_speed;
    `

	if counts := filter.IncludeCounts; counts != nil && *counts {
		query = `
		WITH film_counts AS (
			SELECT 
				f.film_make,
				f.film_type,
				f.id,
				f.film_speed,
				f.color_type,
				f.description,
				f.created,
				f.updated,
				COUNT(p.id) as post_count
			FROM films f
			LEFT JOIN pictures p ON f.film_make = p.film_make AND f.film_type = p.film_type
			GROUP BY f.id, f.film_make, f.film_type, f.film_speed, f.color_type, f.description, f.created, f.updated
		),
		make_totals AS (
			SELECT 
				film_make,
				SUM(post_count) as total_post_count
			FROM film_counts
			GROUP BY film_make
		)
		SELECT 
			fc.id,
			fc.film_make,
			fc.film_type,
			fc.film_speed,
			fc.color_type,
			fc.description,
			fc.created,
			fc.updated,
			fc.post_count,
			mt.total_post_count
		FROM film_counts fc
		JOIN make_totals mt ON fc.film_make = mt.film_make
		ORDER BY fc.film_make, fc.film_type, fc.film_speed;
	`
	}

	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Msg("Find films")
		return nil, err
	}
	defer rows.Close()

	// Map to group film types by make
	makeMap := make(map[string]*analogdb.Film)

	for rows.Next() {
		var id, speed, postCount, totalPostCount int
		var make, filmType, colorType, description string
		var created, updated time.Time

		if err := rows.Scan(
			&id,
			&make,
			&filmType,
			&speed,
			&colorType,
			&description,
			&created,
			&updated,
			&postCount,
			&totalPostCount); err != nil {
			db.logger.Error().Err(err).Ctx(ctx).Msg("Find films - scan error")
			return nil, err
		}

		// Get or create the Film entry (grouped by make)
		if makeMap[make] == nil {
			makeMap[make] = &analogdb.Film{
				Make:      make,
				Types:     []analogdb.FilmType{},
				PostCount: totalPostCount,
			}
		}

		// Add the film type to the make
		filmTypeEntry := analogdb.FilmType{
			Id:          id,
			Make:        make,
			Type:        filmType,
			Speed:       speed,
			ColorType:   colorType,
			Description: description,
			Created:     created,
			Updated:     updated,
			PostCount:   postCount,
		}

		makeMap[make].Types = append(makeMap[make].Types, filmTypeEntry)
	}

	if err = tx.Commit(); err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Msg("Find films")
		return nil, err
	}

	films := make([]*analogdb.Film, 0, len(makeMap))
	for _, film := range makeMap {
		films = append(films, film)
	}

	// Exclude films with no posts?
	if excludeZero := filter.ExcludeZeroCounts; excludeZero != nil && *excludeZero {
		// Only makes sense if we actually included post count.
		if counts := filter.IncludeCounts; counts != nil && *counts {
			films = filterFilmZeroCounts(films)
		}
	}
	return films, nil
}

func filterFilmZeroCounts(films []*analogdb.Film) []*analogdb.Film {
	filtered := make([]*analogdb.Film, 0)
	for _, film := range films {
		if film.PostCount == 0 {
			continue
		}
		// Filter out film types with 0 count
		types := make([]analogdb.FilmType, 0)
		for _, filmType := range film.Types {
			if filmType.PostCount > 0 {
				types = append(types, filmType)
			}
		}
		if len(types) > 0 {
			film.Types = types
			filtered = append(filtered, film)
		}
	}
	return filtered
}

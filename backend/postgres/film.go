package postgres

import (
	"context"
	"database/sql"

	"github.com/evanofslack/analogdb"
)

type FilmService struct {
	db *DB
}

func NewFilmService(db *DB) *FilmService {
	return &FilmService{db: db}
}

func (s *FilmService) Films(ctx context.Context) ([]*analogdb.Film, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	return s.db.findFilms(ctx, tx)
}

// findFilms is the general function responsible for handling all film queries
func (db *DB) findFilms(ctx context.Context, tx *sql.Tx) ([]*analogdb.Film, error) {
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
                updated
            FROM
                films
            ORDER BY
                film_make, film_type, film_speed
	`

	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Msg("Find films")
		return nil, err
	}
	defer rows.Close()

	films := make([]*analogdb.Film, 0)
	for rows.Next() {
	    var f analogdb.Film
		if err := rows.Scan(
			&f.Id,
			&f.Make,
			&f.Type,
			&f.Speed,
			&f.ColorType,
			&f.Description,
			&f.Created,
			&f.Updated); err != nil {
			db.logger.Error().Err(err).Ctx(ctx).Msg("Find films")
			return nil, err
		}
		films = append(films, &f)
	}

	if err = tx.Commit(); err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Msg("Find films")
		return nil, err
	}
	return films, nil
}

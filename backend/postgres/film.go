package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/evanofslack/analogdb"
)

type FilmService struct {
	db *DB
}

func NewFilmService(db *DB) *FilmService {
	return &FilmService{db: db}
}

func (s *FilmService) FindFilms(ctx context.Context, filter *analogdb.FilmFilter) ([]*analogdb.Film, error) {
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
	filterFmt := "nil"
	if filter != nil {
		filterFmt = filter.String()
	}

	db.logger.Debug().Ctx(ctx).Str("filter", filterFmt).Msg("Starting find films")
	defer db.logger.Debug().Ctx(ctx).Str("filter", filterFmt).Msg("Finished find films")

	var args []any
	var where string
	index := 1
	where, args, index = filterToWhereFilm(filter, index)

	order := filterToOrderFilms(filter)
	limit := formatLimitFilms(filter)

	query := fmt.Sprintf(`
		SELECT 
			f.id,
			f.film_make,
			f.film_type,
			f.film_speed,
			f.color_type,
			f.description,
			f.created,
			f.updated,
			0 as post_count
		FROM films f
	    WHERE %s
	    `, where) + order + limit

	if counts := filter.IncludeCounts; counts != nil && *counts {
		having := filterToHavingFilm(filter)
		query = fmt.Sprintf(`
			SELECT 
				f.id,
				f.film_make,
				f.film_type,
				f.film_speed,
				f.color_type,
				f.description,
				f.created,
				f.updated,
				COUNT(p.id) as post_count
			FROM films f
			LEFT JOIN pictures p ON f.film_make = p.film_make AND f.film_type = p.film_type
		    WHERE %s
			GROUP BY f.id, f.film_make, f.film_type, f.film_speed, f.color_type, f.description, f.created, f.updated
			HAVING %s
	`, where, having) + order + limit
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Msg("Find films")
		return nil, err
	}
	defer rows.Close()

	var films []*analogdb.Film

	for rows.Next() {
		var id, speed, postCount int
		var make, filmType, colorType, description string
		var created, updated time.Time

		if err := rows.Scan(&id, &make, &filmType, &speed, &colorType, &description, &created, &updated, &postCount); err != nil {
			db.logger.Error().Err(err).Ctx(ctx).Msg("Find films - scan error")
			return nil, err
		}

		film := &analogdb.Film{
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

		films = append(films, film)
	}

	if err = tx.Commit(); err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Msg("Find films")
		return nil, err
	}
	return films, nil
}

func filterToWhereFilm(filter *analogdb.FilmFilter, startIndex int) (string, []any, int) {
	index := startIndex
	where, args := []string{"1=1"}, []any{}

	if make := filter.Make; make != nil {
		where = append(where, fmt.Sprintf("film_make = $%d", index))
		args = append(args, *make)
		index++
	}
	if ty := filter.Type; ty != nil {
		where = append(where, fmt.Sprintf("film_type = $%d", index))
		args = append(args, *ty)
		index++
	}
	if sp := filter.Speed; sp != nil {
		where = append(where, fmt.Sprintf("film_speed = $%d", index))
		args = append(args, *sp)
		index++
	}
	if color := filter.ColorType; color != nil {
		where = append(where, fmt.Sprintf("color_type = $%d", index))
		args = append(args, *color)
		index++
	}
	if ids := filter.IDs; ids != nil {
		where = append(where, fmt.Sprintf("id = ANY($%d::int[])", index))
		// turn the slice of ids into a string i.e. "(1,2,3)"
		var idsFormat string
		if len(*ids) == 1 {
			// single id can't have a comma
			id := (*ids)[0]
			idsFormat = fmt.Sprintf("{%s}", strconv.Itoa(id))
		} else {
			idsString := []string{}
			for _, i := range *ids {
				idsString = append(idsString, strconv.Itoa(i))
			}
			idsFormat = "{" + strings.Join(idsString, ",") + "}"
		}
		args = append(args, idsFormat)
		index++
	}

	whereQuery := strings.Join(where, " AND ")
	return whereQuery, args, index
}

func filterToHavingFilm(filter *analogdb.FilmFilter) string {
	having := []string{"1=1"}

	if excludeZero := filter.ExcludeZeroCounts; excludeZero != nil && *excludeZero {
		if includeCounts := filter.IncludeCounts; includeCounts != nil && *includeCounts {
			having = append(having, "COUNT(p.id) > 0")
		}
	}

	return strings.Join(having, " AND ")
}

// filterToOrderFilms converts film filter into an SQL "ORDER BY" statement
func filterToOrderFilms(filter *analogdb.FilmFilter) string {
	if sort := filter.Sort; sort != nil {
		switch *sort {
		case analogdb.FilmSortAlphabetically:
			return " ORDER BY f.film_make, f.film_type, f.film_speed DESC"
		case analogdb.FilmSortCounts:
			return " ORDER BY post_count DESC"
		}
	}
	return ""
}

// formatLimitFilms turns the limit into an SQL limit statement
func formatLimitFilms(filter *analogdb.FilmFilter) string {
	if limit := filter.Limit; limit != nil {
		if *limit > 0 {
			return fmt.Sprintf(` LIMIT %d`, *limit)
		}
	}
	return ""
}

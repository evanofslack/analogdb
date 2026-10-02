package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/lib/pq"
)

type FilmService struct {
	db *DB
}

func NewFilmService(db *DB) *FilmService {
	return &FilmService{db: db}
}

func (s *FilmService) FindFilms(ctx context.Context, filter *analogdb.FilmFilter) ([]*analogdb.Film, error) {
	return s.db.findFilms(ctx, filter)
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
		db.logger.ErrorContext(ctx, "Insert film", "error", err, "film_id", id)
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
		db.logger.ErrorContext(ctx, "Insert film", "error", err, "film_id", id)
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	db.logger.InfoContext(ctx, "Finish insert film", "film_id", id)
	film.Id = int(id)

	return film, nil
}

// findFilms is the general function responsible for handling all film queries.
func (db *DB) findFilms(ctx context.Context, filter *analogdb.FilmFilter) ([]*analogdb.Film, error) {
	filterFmt := "nil"
	if filter != nil {
		filterFmt = filter.String()
	}

	db.logger.DebugContext(ctx, "Start find film", "filter", filterFmt)
	defer db.logger.DebugContext(ctx, "Finish find film", "filter", filterFmt)

	var args []any
	var where string
	index := 1
	where, args, index = filterToWhereFilm(filter, index)

	order := filterToOrderFilm(filter)
	limit := formatLimitFilm(filter)

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

	if includeCountsFilm(filter) {
		var countWhere string
		countWhere, args = filterToWhereCountFilm(filter, index, args)
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
				COALESCE(p.post_count, 0) as post_count
			FROM films f
			LEFT JOIN (
				SELECT film_make AS make, film_type AS type, COUNT(*) AS post_count
				FROM pictures
				GROUP BY film_make, film_type
			) p ON p.make = f.film_make AND p.type = f.film_type
		    WHERE %s AND %s
	`, where, countWhere) + order + limit
	}

	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		db.logger.ErrorContext(ctx, "Find films", "error", err)
		return nil, err
	}
	defer rows.Close()

	var films []*analogdb.Film

	for rows.Next() {
		var id, speed, postCount int
		var make, filmType, colorType, description string
		var created, updated time.Time

		if err := rows.Scan(&id, &make, &filmType, &speed, &colorType, &description, &created, &updated, &postCount); err != nil {
			db.logger.ErrorContext(ctx, "Find films, scan error", "error", err)
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
	if err := rows.Err(); err != nil {
		db.logger.ErrorContext(ctx, "Find films", "error", err)
		return nil, err
	}

	if top := filter.TopPosts; top != nil && *top > 0 && len(films) > 0 {
		if err := db.attachFilmTopPosts(ctx, films, *top); err != nil {
			return nil, err
		}
	}

	return films, nil
}

// attachFilmTopPosts adds the highest scoring non nsfw posts to each film
func (db *DB) attachFilmTopPosts(ctx context.Context, films []*analogdb.Film, n int) error {
	makes := make([]string, 0, len(films))
	types := make([]string, 0, len(films))
	for _, f := range films {
		makes = append(makes, f.Make)
		types = append(types, f.Type)
	}

	query := `
		SELECT
			film_make, film_type, id,
			COALESCE(title, ''), COALESCE(score, 0),
			COALESCE(lowurl, ''), COALESCE(lowwidth, 0), COALESCE(lowheight, 0),
			COALESCE(medurl, ''), COALESCE(medwidth, 0), COALESCE(medheight, 0)
		FROM (
			SELECT
				film_make, film_type, id, title, score, lowurl, lowwidth, lowheight, medurl, medwidth, medheight,
				row_number() OVER (PARTITION BY film_make, film_type ORDER BY score DESC NULLS LAST, id DESC) AS rn
			FROM pictures
			WHERE nsfw = false
			AND (film_make, film_type) IN (SELECT * FROM unnest($1::text[], $2::text[]))
		) ranked
		WHERE rn <= $3
		ORDER BY film_make, film_type, rn`

	top, err := db.findCatalogTopPosts(ctx, query, pq.Array(makes), pq.Array(types), n)
	if err != nil {
		db.logger.ErrorContext(ctx, "Find film top posts", "error", err)
		return err
	}
	for _, f := range films {
		f.TopPosts = top[catalogKey(f.Make, f.Type)]
	}
	return nil
}

func includeCountsFilm(filter *analogdb.FilmFilter) bool {
	if counts := filter.IncludeCounts; counts != nil && *counts {
		return true
	}
	return filter.MinCount != nil
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

func filterToWhereCountFilm(filter *analogdb.FilmFilter, index int, args []any) (string, []any) {
	where := []string{"1=1"}
	if excludeZero := filter.ExcludeZeroCounts; excludeZero != nil && *excludeZero {
		if includeCounts := filter.IncludeCounts; includeCounts != nil && *includeCounts {
			where = append(where, "COALESCE(p.post_count, 0) > 0")
		}
	}
	if minCount := filter.MinCount; minCount != nil {
		where = append(where, fmt.Sprintf("COALESCE(p.post_count, 0) >= $%d", index))
		args = append(args, *minCount)
	}
	return strings.Join(where, " AND "), args
}

// filterToOrderFilm converts film filter into an SQL "ORDER BY" statement
func filterToOrderFilm(filter *analogdb.FilmFilter) string {
	if sort := filter.Sort; sort != nil {
		switch *sort {
		case analogdb.FilmSortAlphabetical:
			return " ORDER BY f.film_make, f.film_type, f.film_speed"
		case analogdb.FilmSortCounts:
			return " ORDER BY post_count DESC"
		}
	}
	return ""
}

// formatLimitFilm turns the limit into an SQL limit statement
func formatLimitFilm(filter *analogdb.FilmFilter) string {
	if limit := filter.Limit; limit != nil {
		if *limit > 0 {
			return fmt.Sprintf(` LIMIT %d`, *limit)
		}
	}
	return ""
}

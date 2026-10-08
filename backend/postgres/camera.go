package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/lib/pq"
)

type CameraService struct {
	db *DB
}

func NewCameraService(db *DB) *CameraService {
	return &CameraService{db: db}
}

func (s *CameraService) FindCameras(ctx context.Context, filter *analogdb.CameraFilter) ([]*analogdb.Camera, error) {
	return s.db.findCameras(ctx, filter)
}

func (s *CameraService) CreateCamera(ctx context.Context, camera *analogdb.CreateCamera) (*analogdb.CreateCamera, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	created, err := s.db.createCamera(ctx, tx, camera)
	if err != nil {
		return nil, err
	}
	return created, nil
}

// DeleteCamera deletes a catalog camera. It refuses while any post still uses its
// make and model, so a delete never leaves posts on a name the catalog lacks.
func (s *CameraService) DeleteCamera(ctx context.Context, id int) error {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var make, name string
	err = tx.QueryRowContext(ctx,
		`SELECT camera_make, camera_model FROM cameras WHERE id = $1 FOR UPDATE`, id,
	).Scan(&make, &name)
	if errors.Is(err, sql.ErrNoRows) {
		return &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "camera not found"}
	} else if err != nil {
		return err
	}

	var posts int
	err = tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pictures WHERE camera_make = $1 AND camera_model = $2`, make, name,
	).Scan(&posts)
	if err != nil {
		return err
	}
	if posts > 0 {
		return &analogdb.Error{
			Code:    analogdb.ERRCONFLICT,
			Message: fmt.Sprintf("camera %s %s is used by %d posts", make, name, posts),
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM cameras WHERE id = $1`, id); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.db.logger.InfoContext(ctx, "Deleted camera", "camera_id", id, "make", make, "name", name)
	return nil
}

func (db *DB) createCamera(ctx context.Context, tx *sql.Tx, camera *analogdb.CreateCamera) (*analogdb.CreateCamera, error) {
	var id int64

	query := `
	INSERT INTO cameras
        (camera_make, camera_model, description)
        VALUES ($1, $2, $3)
        ON CONFLICT (camera_make, camera_model) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            updated = CURRENT_TIMESTAMP
        RETURNING id
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		db.logger.ErrorContext(ctx, "Insert camera", "error", err, "camera_id", id)
		return nil, err
	}
	defer stmt.Close()

	err = stmt.QueryRowContext(
		ctx,
		camera.Make,
		camera.Model,
		camera.Description).Scan(&id)
	if err != nil {
		db.logger.ErrorContext(ctx, "Insert camera", "error", err, "camera_id", id)
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	db.logger.InfoContext(ctx, "Finished inserting camera", "camera_id", id)
	camera.Id = int(id)

	return camera, nil
}

// findCameras is the general function responsible for handling all camera queries
func (db *DB) findCameras(ctx context.Context, filter *analogdb.CameraFilter) ([]*analogdb.Camera, error) {
	filterFmt := "nil"
	if filter != nil {
		filterFmt = filter.String()
	}
	db.logger.DebugContext(ctx, "Start find cameras", "filter", filterFmt)
	defer db.logger.DebugContext(ctx, "Finish find cameras", "filter", filterFmt)

	var args []any
	var where string
	index := 1
	where, args, index = filterToWhereCamera(filter, index)

	order := filterToOrderCamera(filter)
	limit := formatLimitCamera(filter)

	query := fmt.Sprintf(`
		SELECT 
			c.id,
			c.camera_make,
			c.camera_model,
			c.description,
			c.created,
			c.updated,
			0 as post_count
		FROM cameras c
	    WHERE %s
		`, where) + order + limit

	if includeCountsCamera(filter) {
		var countWhere string
		countWhere, args = filterToWhereCountCamera(filter, index, args)
		query = fmt.Sprintf(`
			SELECT
				c.id,
				c.camera_make,
				c.camera_model,
				c.description,
				c.created,
				c.updated,
				COALESCE(p.post_count, 0) as post_count
			FROM cameras c
			LEFT JOIN (
				SELECT camera_make AS make, camera_model AS model, COUNT(*) AS post_count
				FROM pictures
				GROUP BY camera_make, camera_model
			) p ON p.make = c.camera_make AND p.model = c.camera_model
		    WHERE %s AND %s
			`, where, countWhere) + order + limit
	}

	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		db.logger.ErrorContext(ctx, "Find cameras", "error", err)
		return nil, err
	}
	defer rows.Close()

	var cameras []*analogdb.Camera

	for rows.Next() {
		var id, postCount int
		var make, model, description string
		var created, updated time.Time

		if err := rows.Scan(&id, &make, &model, &description, &created, &updated, &postCount); err != nil {
			db.logger.ErrorContext(ctx, "Find cameras", "error", err)
			return nil, err
		}

		camera := &analogdb.Camera{
			Id:          id,
			Make:        make,
			Model:       model,
			Description: description,
			Created:     created,
			Updated:     updated,
			PostCount:   postCount,
		}

		cameras = append(cameras, camera)
	}
	if err := rows.Err(); err != nil {
		db.logger.ErrorContext(ctx, "Find cameras", "error", err)
		return nil, err
	}

	if excludeZero := filter.ExcludeZeroCounts; excludeZero != nil && *excludeZero {
		if counts := filter.IncludeCounts; counts != nil && *counts {
			cameras = filterCameraZeroCounts(cameras)
		}
	}

	if top := filter.TopPosts; top != nil && *top > 0 && len(cameras) > 0 {
		if err := db.attachCameraTopPosts(ctx, cameras, *top); err != nil {
			return nil, err
		}
	}

	return cameras, nil
}

// attachCameraTopPosts adds the highest scoring non nsfw posts to each camera
func (db *DB) attachCameraTopPosts(ctx context.Context, cameras []*analogdb.Camera, n int) error {
	makes := make([]string, 0, len(cameras))
	models := make([]string, 0, len(cameras))
	for _, c := range cameras {
		makes = append(makes, c.Make)
		models = append(models, c.Model)
	}

	query := `
		SELECT
			camera_make, camera_model, id,
			COALESCE(title, ''), COALESCE(score, 0),
			COALESCE(lowurl, ''), COALESCE(lowwidth, 0), COALESCE(lowheight, 0),
			COALESCE(medurl, ''), COALESCE(medwidth, 0), COALESCE(medheight, 0)
		FROM (
			SELECT
				camera_make, camera_model, id, title, score, lowurl, lowwidth, lowheight, medurl, medwidth, medheight,
				row_number() OVER (PARTITION BY camera_make, camera_model ORDER BY score DESC NULLS LAST, id DESC) AS rn
			FROM pictures
			WHERE nsfw = false
			AND (camera_make, camera_model) IN (SELECT * FROM unnest($1::text[], $2::text[]))
		) ranked
		WHERE rn <= $3
		ORDER BY camera_make, camera_model, rn`

	top, err := db.findCatalogTopPosts(ctx, query, pq.Array(makes), pq.Array(models), n)
	if err != nil {
		db.logger.ErrorContext(ctx, "Find camera top posts", "error", err)
		return err
	}
	for _, c := range cameras {
		c.TopPosts = top[catalogKey(c.Make, c.Model)]
	}
	return nil
}

func includeCountsCamera(filter *analogdb.CameraFilter) bool {
	if counts := filter.IncludeCounts; counts != nil && *counts {
		return true
	}
	return filter.MinCount != nil
}

func filterCameraZeroCounts(cameras []*analogdb.Camera) []*analogdb.Camera {
	filtered := make([]*analogdb.Camera, 0)
	for _, camera := range cameras {
		if camera.PostCount > 0 {
			filtered = append(filtered, camera)
		}
	}
	return filtered
}

func filterToWhereCamera(filter *analogdb.CameraFilter, startIndex int) (string, []any, int) {
	index := startIndex
	where, args := []string{"1=1"}, []any{}

	if make := filter.Make; make != nil {
		where = append(where, fmt.Sprintf("camera_make = $%d", index))
		args = append(args, *make)
		index++
	}
	if ty := filter.Model; ty != nil {
		where = append(where, fmt.Sprintf("camera_model = $%d", index))
		args = append(args, *ty)
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

func filterToWhereCountCamera(filter *analogdb.CameraFilter, index int, args []any) (string, []any) {
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

// filterToOrderCamera converts camera filter into an SQL "ORDER BY" statement
func filterToOrderCamera(filter *analogdb.CameraFilter) string {
	if sort := filter.Sort; sort != nil {
		switch *sort {
		case analogdb.CameraSortAlphabetical:
			return " ORDER BY c.camera_make, c.camera_model"
		case analogdb.CameraSortCounts:
			return " ORDER BY post_count DESC"
		}
	}
	return ""
}

// formatLimitCamera turns the limit into an SQL limit statement
func formatLimitCamera(filter *analogdb.CameraFilter) string {
	if limit := filter.Limit; limit != nil {
		if *limit > 0 {
			return fmt.Sprintf(` LIMIT %d`, *limit)
		}
	}
	return ""
}

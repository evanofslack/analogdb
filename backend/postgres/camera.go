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

type CameraService struct {
	db *DB
}

func NewCameraService(db *DB) *CameraService {
	return &CameraService{db: db}
}

func (s *CameraService) FindCameras(ctx context.Context, filter *analogdb.CameraFilter) ([]*analogdb.Camera, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	return s.db.findCameras(ctx, tx, filter)
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
		db.logger.Error().Ctx(ctx).Err(err).Int64("camera_id", id).Msg("Insert camera")
		return nil, err
	}
	defer stmt.Close()

	err = stmt.QueryRowContext(
		ctx,
		camera.Make,
		camera.Model,
		camera.Description).Scan(&id)
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int64("camera_id", id).Msg("Insert camera")
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	db.logger.Info().Ctx(ctx).Int64("postID", id).Msg("Finished inserting camera")
	camera.Id = int(id)

	return camera, nil
}

// findCameras is the general function responsible for handling all camera queries
func (db *DB) findCameras(ctx context.Context, tx *sql.Tx, filter *analogdb.CameraFilter) ([]*analogdb.Camera, error) {
	filterFmt := "nil"
	if filter != nil {
		filterFmt = filter.String()
	}
	db.logger.Debug().Ctx(ctx).Str("filter", filterFmt).Msg("Starting find cameras")
	defer db.logger.Debug().Ctx(ctx).Str("filter", filterFmt).Msg("Finished find cameras")

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

	if counts := filter.IncludeCounts; counts != nil && *counts {
		having := filterToHavingCamera(filter)
		query = fmt.Sprintf(`
			SELECT 
				c.id,
				c.camera_make,
				c.camera_model,
				c.description,
				c.created,
				c.updated,
				COUNT(p.id) as post_count
			FROM cameras c
			LEFT JOIN pictures p ON c.camera_make = p.camera_make AND c.camera_model = p.camera_model
		    WHERE %s
			GROUP BY c.id, c.camera_make, c.camera_model, c.description, c.created, c.updated
		    HAVING %s
			`, where, having) + order + limit
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Msg("Find cameras")
		return nil, err
	}
	defer rows.Close()

	var cameras []*analogdb.Camera

	for rows.Next() {
		var id, postCount int
		var make, model, description string
		var created, updated time.Time

		if err := rows.Scan(&id, &make, &model, &description, &created, &updated, &postCount); err != nil {
			db.logger.Error().Err(err).Ctx(ctx).Msg("Find cameras")
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

	if err = tx.Commit(); err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Msg("Find cameras")
		return nil, err
	}

	if excludeZero := filter.ExcludeZeroCounts; excludeZero != nil && *excludeZero {
		if counts := filter.IncludeCounts; counts != nil && *counts {
			cameras = filterCameraZeroCounts(cameras)
		}
	}

	return cameras, nil
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

func filterToHavingCamera(filter *analogdb.CameraFilter) string {
	having := []string{"1=1"}
	if excludeZero := filter.ExcludeZeroCounts; excludeZero != nil && *excludeZero {
		if includeCounts := filter.IncludeCounts; includeCounts != nil && *includeCounts {
			having = append(having, "COUNT(p.id) > 0")
		}
	}
	return strings.Join(having, " AND ")
}

// filterToOrderCamera converts camera filter into an SQL "ORDER BY" statement
func filterToOrderCamera(filter *analogdb.CameraFilter) string {
	if sort := filter.Sort; sort != nil {
		switch *sort {
		case analogdb.CameraSortAlphabetical:
			return " ORDER BY c.camera_make, c.camera_model DESC"
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

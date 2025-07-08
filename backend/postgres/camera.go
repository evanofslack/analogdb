package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/evanofslack/analogdb"
)

type CameraService struct {
	db *DB
}

func NewCameraService(db *DB) *CameraService {
	return &CameraService{db: db}
}

func (s *CameraService) AllCameras(ctx context.Context, filter *analogdb.CameraFilter) ([]*analogdb.Camera, error) {
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
        ON CONFLICT (camera_make, camera_model) DO NOTHING
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
	db.logger.Debug().Ctx(ctx).Msg("Starting find cameras")
	defer db.logger.Debug().Ctx(ctx).Msg("Finished find cameras")

	query := `
    SELECT 
    id,
    camera_make,
    camera_model,
    description,
    created,
    updated,
    0 as post_count,
    0 as total_post_count
FROM cameras
ORDER BY camera_make, camera_model;
    `

	if counts := filter.IncludeCounts; counts != nil && *counts {
		query = `
		WITH camera_counts AS (
			SELECT 
				c.camera_make,
				c.camera_model,
				c.id,
				c.description,
				c.created,
				c.updated,
				COUNT(p.id) as post_count
			FROM cameras c
			LEFT JOIN pictures p ON c.camera_make = p.camera_make AND c.camera_model = p.camera_model
			GROUP BY c.id, c.camera_make, c.camera_model, c.description, c.created, c.updated
		),
		make_totals AS (
			SELECT 
				camera_make,
				SUM(post_count) as total_post_count
			FROM camera_counts
			GROUP BY camera_make
		)
		SELECT 
			cc.id,
			cc.camera_make,
			cc.camera_model,
			cc.description,
			cc.created,
			cc.updated,
			cc.post_count,
			mt.total_post_count
		FROM camera_counts cc
		JOIN make_totals mt ON cc.camera_make = mt.camera_make
		ORDER BY cc.camera_make, cc.camera_model;
	`
	}

	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Msg("Find cameras")
		return nil, err
	}
	defer rows.Close()

	// Map to group camera models by make
	makeMap := make(map[string]*analogdb.Camera)

	for rows.Next() {
		var id, postCount, totalPostCount int
		var make, model, description string
		var created, updated time.Time
		if err := rows.Scan(
			&id,
			&make,
			&model,
			&description,
			&created,
			&updated,
			&postCount,
			&totalPostCount); err != nil {
			db.logger.Error().Err(err).Ctx(ctx).Msg("Find cameras")
			return nil, err
		}

		// Get or create the Camera entry (grouped by make)
		if makeMap[make] == nil {
			makeMap[make] = &analogdb.Camera{
				Make:      make,
				Models:    []analogdb.CameraModel{},
				PostCount: totalPostCount,
			}
		}

		// Add the camera model to the make
		cameraModelEntry := analogdb.CameraModel{
			Id:          id,
			Make:        make,
			Model:       model,
			Description: description,
			Created:     created,
			Updated:     updated,
			PostCount:   postCount,
		}

		makeMap[make].Models = append(makeMap[make].Models, cameraModelEntry)
	}

	if err = tx.Commit(); err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Msg("Find cameras")
		return nil, err
	}

	cameras := make([]*analogdb.Camera, 0, len(makeMap))
	for _, camera := range makeMap {
		cameras = append(cameras, camera)
	}

	// Exclude cameras with no posts?
	if excludeZero := filter.ExcludeZeroCounts; excludeZero != nil && *excludeZero {
		// Only makes sense if we actually included post count.
		if counts := filter.IncludeCounts; counts != nil && *counts {
			cameras = filterCameraZeroCounts(cameras)
		}
	}
	return cameras, nil
}

func filterCameraZeroCounts(cameras []*analogdb.Camera) []*analogdb.Camera {
	filtered := make([]*analogdb.Camera, 0)
	for _, camera := range cameras {
		if camera.PostCount == 0 {
			continue
		}
		// Filter out film types with 0 count
		models := make([]analogdb.CameraModel, 0)
		for _, model := range camera.Models {
			if model.PostCount > 0 {
				models = append(models, model)
			}
		}
		if len(models) > 0 {
			camera.Models = models
			filtered = append(filtered, camera)
		}
	}
	return filtered
}

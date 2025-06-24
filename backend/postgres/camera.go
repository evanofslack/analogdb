package postgres

import (
	"context"
	"database/sql"

	"github.com/evanofslack/analogdb"
)

type CameraService struct {
	db *DB
}

func NewCameraService(db *DB) *CameraService {
	return &CameraService{db: db}
}

func (s *CameraService) AllCameras(ctx context.Context) ([]*analogdb.Camera, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	return s.db.findCameras(ctx, tx)
}

func (s *CameraService) CreateCamera(ctx context.Context, camera *analogdb.Camera) (*analogdb.Camera, error) {
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

func (db *DB) createCamera(ctx context.Context, tx *sql.Tx, camera *analogdb.Camera) (*analogdb.Camera, error) {
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
func (db *DB) findCameras(ctx context.Context, tx *sql.Tx) ([]*analogdb.Camera, error) {
	db.logger.Debug().Ctx(ctx).Msg("Starting find cameras")
	defer db.logger.Debug().Ctx(ctx).Msg("Finished find cameras")

	query := `
			SELECT
                id,
                camera_make,
                camera_model,
                description,
                created,
                updated
            FROM
                cameras
            ORDER BY
                camera_make, camera_model
	`

	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Msg("Find cameras")
		return nil, err
	}
	defer rows.Close()

	cameras := make([]*analogdb.Camera, 0)
	for rows.Next() {
		var c analogdb.Camera
		if err := rows.Scan(
			&c.Id,
			&c.Make,
			&c.Model,
			&c.Description,
			&c.Created,
			&c.Updated); err != nil {
			db.logger.Error().Err(err).Ctx(ctx).Msg("Find cameras")
			return nil, err
		}
		cameras = append(cameras, &c)
	}

	if err = tx.Commit(); err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Msg("Find cameras")
		return nil, err
	}
	return cameras, nil
}

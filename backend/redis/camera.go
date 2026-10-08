package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/mitchellh/hashstructure/v2"

	"github.com/evanofslack/analogdb"
)

const (
	// name of camera cache
	cameraInstance = "camera"
	// ttl for individual camera in cache
	cameraTTL = time.Hour * 24
	// im memory cache size for individual cameras
	cameraLocalSize = 100
)

// ensure interface is implemented
var _ analogdb.CameraService = (*CameraService)(nil)

type CameraService struct {
	rdb         *RDB
	cameraCache *Cache
	dbService   analogdb.CameraService
}

func NewCacheCameraService(rdb *RDB, dbService analogdb.CameraService) *CameraService {
	cameraCache := rdb.NewCache(cameraInstance, cameraLocalSize, cameraTTL)

	return &CameraService{
		rdb:         rdb,
		cameraCache: cameraCache,
		dbService:   dbService,
	}
}

func (s *CameraService) CreateCamera(ctx context.Context, camera *analogdb.CreateCamera) (*analogdb.CreateCamera, error) {
	created, err := s.dbService.CreateCamera(ctx, camera)
	if err != nil {
		return nil, err
	}
	s.rdb.bumpGen(ctx, camerasEntity)
	return created, nil
}

func (s *CameraService) DeleteCamera(ctx context.Context, id int) error {
	if err := s.dbService.DeleteCamera(ctx, id); err != nil {
		return err
	}
	s.rdb.bumpGen(ctx, camerasEntity)
	return nil
}

func (s *CameraService) FindCameras(ctx context.Context, filter *analogdb.CameraFilter) ([]*analogdb.Camera, error) {
	s.rdb.logger.DebugContext(ctx, "Start find cameras with cache", "instance", s.cameraCache.instance)
	defer s.rdb.logger.DebugContext(ctx, "Finish find cameras with cache", "instance", s.cameraCache.instance)

	// generate a unique hash from the filter struct
	hash, err := hashstructure.Hash(filter, hashstructure.FormatV2, nil)
	if err != nil {
		s.rdb.logger.ErrorContext(ctx, "Fail hash camera filter", "instance", s.cameraCache.instance, "error", err)

		// if we failed, fallback to db
		return s.dbService.FindCameras(ctx, filter)
	}

	camerasKey := s.rdb.genCacheKey(ctx, camerasEntity, fmt.Sprint(hash))

	return fetch(ctx, s.cameraCache, camerasKey, cameraTTL, func(ctx context.Context) ([]*analogdb.Camera, error) {
		return s.dbService.FindCameras(ctx, filter)
	})
}

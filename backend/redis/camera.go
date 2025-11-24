package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/cache/v9"
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
	return s.dbService.CreateCamera(ctx, camera)
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

	camerasHash := fmt.Sprint(hash)

	var cameras []*analogdb.Camera

	// try to get cameras from cache
	camerasErr := s.cameraCache.get(ctx, camerasHash, &cameras)

	// no error means we found in cache
	if camerasErr == nil {
		return cameras, nil
	}

	// fallback to db
	cameras, err = s.dbService.FindCameras(ctx, filter)
	if err != nil {
		return nil, err
	}

	// add cameras to cache
	// do this async so response is returned quicker
	go func() {
		s.rdb.logger.DebugContext(ctx, "Add cameras to cache", "instance", s.cameraCache.instance)

		// create a new context; orignal one will be canceled when request is closed
		ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
		defer cancel()

		// add cameras to cache
		if err := s.cameraCache.set(ctx, &cache.Item{
			Ctx:   ctx,
			Key:   camerasHash,
			Value: &cameras,
			TTL:   cameraTTL,
		}); err != nil {
			s.rdb.logger.ErrorContext(ctx, "Fail add cameras to cache", "instance", s.cameraCache.instance, "error", err)
		}
	}()

	return cameras, nil
}

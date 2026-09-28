package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/mitchellh/hashstructure/v2"

	"github.com/evanofslack/analogdb"
)

const (
	// name of film cache
	filmInstance = "film"
	// ttl for individual film in cache
	filmTTL = time.Hour * 24
	// im memory cache size for individual films
	filmLocalSize = 100
)

// ensure interface is implemented
var _ analogdb.FilmService = (*FilmService)(nil)

type FilmService struct {
	rdb       *RDB
	filmCache *Cache
	dbService analogdb.FilmService
}

func NewCacheFilmService(rdb *RDB, dbService analogdb.FilmService) *FilmService {
	filmCache := rdb.NewCache(filmInstance, filmLocalSize, filmTTL)

	return &FilmService{
		rdb:       rdb,
		filmCache: filmCache,
		dbService: dbService,
	}
}

func (s *FilmService) CreateFilm(ctx context.Context, film *analogdb.CreateFilm) (*analogdb.CreateFilm, error) {
	created, err := s.dbService.CreateFilm(ctx, film)
	if err != nil {
		return nil, err
	}
	s.rdb.bumpGen(ctx, filmsEntity)
	return created, nil
}

func (s *FilmService) FindFilms(ctx context.Context, filter *analogdb.FilmFilter) ([]*analogdb.Film, error) {
	s.rdb.logger.DebugContext(ctx, "Start find films with cache", "instance", s.filmCache.instance)
	defer s.rdb.logger.DebugContext(ctx, "Finish find films with cache", "instance", s.filmCache.instance)

	// generate a unique hash from the filter struct
	hash, err := hashstructure.Hash(filter, hashstructure.FormatV2, nil)
	if err != nil {
		s.rdb.logger.ErrorContext(ctx, "Fail hash film filter", "instance", s.filmCache.instance, "error", err)

		// if we failed, fallback to db
		return s.dbService.FindFilms(ctx, filter)
	}

	filmsKey := s.rdb.genCacheKey(ctx, filmsEntity, fmt.Sprint(hash))

	return fetch(ctx, s.filmCache, filmsKey, filmTTL, func(ctx context.Context) ([]*analogdb.Film, error) {
		return s.dbService.FindFilms(ctx, filter)
	})
}

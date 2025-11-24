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
	return s.dbService.CreateFilm(ctx, film)
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

	filmsHash := fmt.Sprint(hash)

	var films []*analogdb.Film

	// try to get films from cache
	filmsErr := s.filmCache.get(ctx, filmsHash, &films)

	// no error means we found in cache
	if filmsErr == nil {
		return films, nil
	}

	// fallback to db
	films, err = s.dbService.FindFilms(ctx, filter)
	if err != nil {
		return nil, err
	}

	// add films to cache
	// do this async so response is returned quicker
	go func() {
		s.rdb.logger.DebugContext(ctx, "Add film to cache", "instance", s.filmCache.instance)

		// create a new context; orignal one will be canceled when request is closed
		ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
		defer cancel()

		// add films to cache
		if err := s.filmCache.set(ctx, &cache.Item{
			Ctx:   ctx,
			Key:   filmsHash,
			Value: &films,
			TTL:   filmTTL,
		}); err != nil {
			s.rdb.logger.ErrorContext(ctx, "Fail add film to cache", "instance", s.filmCache.instance, "error", err)
		}
	}()

	return films, nil
}

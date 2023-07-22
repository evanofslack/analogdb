package redis

import (
	"context"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/go-redis/cache/v8"
)

const (
	authorsInstance  = "authors"
	authorsLocalSize = 1000
	authorsTTL       = time.Hour * 4
	authorsKey       = "authors"
)

// ensure interface is implemented
var _ analogdb.AuthorService = (*AuthorService)(nil)

type AuthorService struct {
	rdb       *RDB
	cache     *Cache
	dbService analogdb.AuthorService
}

func NewCacheAuthorService(rdb *RDB, dbService analogdb.AuthorService) *AuthorService {

	cache := rdb.NewCache(authorsInstance, authorsLocalSize, authorsTTL)

	return &AuthorService{
		rdb:       rdb,
		cache:     cache,
		dbService: dbService,
	}
}

func (s *AuthorService) FindAuthors(ctx context.Context) ([]string, error) {

	s.rdb.logger.Debug().Str("instance", s.cache.instance).Msg("Starting find authors with cache")
	defer func() {
		s.rdb.logger.Debug().Str("instance", s.cache.instance).Msg("Finished find authors with cache")
	}()

	var authors []string

	// try to get from the cache
	err := s.cache.get(ctx, authorsKey, &authors)

	// no error means we found it
	if err == nil {
		return authors, nil
	}

	// fallback to postgres if not in cache
	authors, err = s.dbService.FindAuthors(ctx)
	if err != nil {
		return nil, err
	}

	// add to cache
	// do this async so response is returned quicker
	go func() {

		s.rdb.logger.Debug().Str("instance", s.cache.instance).Msg("Adding authors to cache")

		// create a new context; orignal one will be canceled when request is closed
		ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
		defer cancel()

		s.cache.set(&cache.Item{
			Ctx:   ctx,
			Key:   authorsKey,
			Value: &authors,
			TTL:   authorsTTL,
		})
	}()

	return authors, nil
}

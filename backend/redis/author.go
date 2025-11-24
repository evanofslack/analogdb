package redis

import (
	"context"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/go-redis/cache/v9"
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
	s.rdb.logger.DebugContext(ctx, "Start find authors with cache", "instance", s.cache.instance)
	defer func() {
		s.rdb.logger.DebugContext(ctx, "Finish find authors with cache", "instance", s.cache.instance)
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
		s.rdb.logger.DebugContext(ctx, "Add authors to cache", "instance", s.cache.instance)

		// create a new context; orignal one will be canceled when request is closed
		ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
		defer cancel()

		s.cache.set(ctx, &cache.Item{
			Ctx:   ctx,
			Key:   authorsKey,
			Value: &authors,
			TTL:   authorsTTL,
		})
	}()

	return authors, nil
}

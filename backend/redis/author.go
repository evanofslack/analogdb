package redis

import (
	"context"
	"strings"
	goTime "time"

	"github.com/evanofslack/analogdb"
	"github.com/go-redis/cache/v8"
)

const (
	authorsTTL = goTime.Hour * 4
	authorsKey = "authors"
)

// ensure interface is implemented
var _ analogdb.AuthorService = (*AuthorService)(nil)

type AuthorService struct {
	rdb       *RDB
	cache     *cache.Cache
	dbService analogdb.AuthorService
}

func NewCacheAuthorService(rdb *RDB, dbService analogdb.AuthorService) *AuthorService {
	cache := cache.New(&cache.Options{
		Redis:      rdb.db,
		LocalCache: cache.NewTinyLFU(1000, authorsTTL),
	})

	return &AuthorService{
		rdb:       rdb,
		cache:     cache,
		dbService: dbService,
	}
}

func (s *AuthorService) FindAuthors(ctx context.Context) ([]string, error) {

	s.rdb.logger.Debug().Msg("Getting authors with cache")

	var cachedAuthors string

	// try to get from the cache
	err := s.cache.Get(ctx, authorsKey, &cachedAuthors)

	// we found in cache, return
	if cachedAuthors != "" && err == nil {
		s.rdb.logger.Debug().Msg("Found authors in cache")
		authors := strings.Split(cachedAuthors, ",")
		return authors, nil
	}

	s.rdb.logger.Debug().Msg("Failed to get authors from cache (cache miss)")
	if err != nil {
		s.rdb.logger.Info().Str("error", err.Error()).Msg("Failed to get authors from cache")
	}

	// fallback to postgres if not in cache
	authors, err := s.dbService.FindAuthors(ctx)
	if err != nil {
		return nil, err
	}

	authorsString := strings.Join(authors, ",")

	// add to cache
	if err := s.cache.Set(&cache.Item{
		Ctx:   ctx,
		Key:   authorsKey,
		Value: &authorsString,
		TTL:   authorsTTL,
	}); err != nil {
		s.rdb.logger.Err(err).Msg("Failed to add authors to cache")
	}

	s.rdb.logger.Debug().Msg("Added authors to cache")

	return authors, nil
}

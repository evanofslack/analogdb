package redis

import (
	"context"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/go-redis/cache/v8"
)

const (
	authorsTTL = time.Hour * 4
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

	s.rdb.logger.Debug().Msg("Starting find authors with cache")
	defer func() {
		s.rdb.logger.Debug().Msg("Finished find authors with cache")
	}()

	var authors []string

	// try to get from the cache
	err := s.cache.Get(ctx, authorsKey, &authors)
	if err == nil {
		s.rdb.logger.Debug().Msg("Found authors in cache")
		return authors, nil
	}

	// fallback to postgres if not in cache
	s.rdb.logger.Info().Str("error", err.Error()).Msg("Failed to get authors from cache")
	authors, err = s.dbService.FindAuthors(ctx)
	if err != nil {
		return nil, err
	}

	// add to cache
	// do this async so response is returned quicker
	go func() {

		s.rdb.logger.Debug().Msg("Adding authors to cache")
		// create a new context; orignal one will be canceled when request is closed
		ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
		defer cancel()

		if err := s.cache.Set(&cache.Item{
			Ctx:   ctx,
			Key:   authorsKey,
			Value: &authors,
			TTL:   authorsTTL,
		}); err != nil {
			s.rdb.logger.Err(err).Msg("Failed to add authors to cache")
		} else {
			s.rdb.logger.Debug().Msg("Added authors to cache")
		}
	}()

	return authors, nil
}

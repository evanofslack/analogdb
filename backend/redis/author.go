package redis

import (
	"context"
	"time"

	"github.com/evanofslack/analogdb"
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

	key := s.rdb.genCacheKey(ctx, postsEntity, authorsKey)

	return fetch(ctx, s.cache, key, authorsTTL, func(ctx context.Context) ([]string, error) {
		return s.dbService.FindAuthors(ctx)
	})
}

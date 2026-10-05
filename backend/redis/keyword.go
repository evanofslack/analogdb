package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/mitchellh/hashstructure/v2"

	"github.com/evanofslack/analogdb"
)

const (
	keywordInstance  = "keyword"
	keywordLocalSize = 1000
	keywordTTL       = time.Hour
)

// ensure interface is implemented
var _ analogdb.KeywordService = (*KeywordService)(nil)

type KeywordService struct {
	rdb          *RDB
	keywordCache *Cache
	dbService    analogdb.KeywordService
}

func NewCacheKeywordService(rdb *RDB, dbService analogdb.KeywordService) *KeywordService {
	keywordCache := rdb.NewCache(keywordInstance, keywordLocalSize, keywordTTL)

	return &KeywordService{
		rdb:          rdb,
		keywordCache: keywordCache,
		dbService:    dbService,
	}
}

func (s *KeywordService) GetKeywordSummary(ctx context.Context, filter *analogdb.KeywordFilter) (*[]analogdb.KeywordSummary, error) {
	hash, err := hashstructure.Hash(filter, hashstructure.FormatV2, nil)
	if err != nil {
		s.rdb.logger.ErrorContext(ctx, "Fail hash keyword filter", "instance", s.keywordCache.instance, "error", err)
		return s.dbService.GetKeywordSummary(ctx, filter)
	}

	summary, err := fetch(ctx, s.keywordCache, fmt.Sprintf("keywords:summary:%d", hash), keywordTTL, func(ctx context.Context) ([]analogdb.KeywordSummary, error) {
		summary, err := s.dbService.GetKeywordSummary(ctx, filter)
		if err != nil {
			return nil, err
		}
		return *summary, nil
	})
	if err != nil {
		return nil, err
	}
	return &summary, nil
}

// TagCounts is not cached here, the server keeps the counts in memory
func (s *KeywordService) TagCounts(ctx context.Context) (map[string]int, int, error) {
	return s.dbService.TagCounts(ctx)
}

func (s *KeywordService) FindKeyword(ctx context.Context, word string, topPosts int) (*analogdb.KeywordDetail, error) {
	key := fmt.Sprintf("keywords:detail:%d:%s", topPosts, word)
	return fetch(ctx, s.keywordCache, key, keywordTTL, func(ctx context.Context) (*analogdb.KeywordDetail, error) {
		return s.dbService.FindKeyword(ctx, word, topPosts)
	})
}

func (s *KeywordService) CoOccurring(ctx context.Context, word string) (map[string]int, error) {
	return fetch(ctx, s.keywordCache, "keywords:related:"+word, keywordTTL, func(ctx context.Context) (map[string]int, error) {
		return s.dbService.CoOccurring(ctx, word)
	})
}

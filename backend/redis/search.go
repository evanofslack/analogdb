package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/mitchellh/hashstructure/v2"

	"github.com/evanofslack/analogdb"
)

const (
	searchInstance  = "search"
	searchLocalSize = 1000
	searchTTL       = time.Minute * 10
)

// ensure interface is implemented
var _ analogdb.SearchService = (*SearchService)(nil)

type SearchService struct {
	rdb         *RDB
	searchCache *Cache
	dbService   analogdb.SearchService
}

// searchPage is the cached result of one text search page
type searchPage struct {
	Hits    []analogdb.SearchHit
	HasMore bool
}

// searchKey holds the filter fields that change text search results
type searchKey struct {
	Query     string
	Expanded  string
	Nsfw      *bool
	Grayscale *bool
	Sprocket  *bool
	Limit     int
	Offset    int
}

func NewCacheSearchService(rdb *RDB, dbService analogdb.SearchService) *SearchService {
	searchCache := rdb.NewCache(searchInstance, searchLocalSize, searchTTL)

	return &SearchService{
		rdb:         rdb,
		searchCache: searchCache,
		dbService:   dbService,
	}
}

func (s *SearchService) SearchText(ctx context.Context, filter *analogdb.SearchFilter) ([]analogdb.SearchHit, bool, error) {
	key := searchKey{
		Query:     filter.Query,
		Expanded:  filter.Expanded,
		Nsfw:      filter.Nsfw,
		Grayscale: filter.Grayscale,
		Sprocket:  filter.Sprocket,
		Limit:     filter.Limit,
		Offset:    filter.Offset,
	}
	hash, err := hashstructure.Hash(key, hashstructure.FormatV2, nil)
	if err != nil {
		s.rdb.logger.ErrorContext(ctx, "Fail hash search filter", "instance", s.searchCache.instance, "error", err)
		return s.dbService.SearchText(ctx, filter)
	}

	page, err := fetch(ctx, s.searchCache, fmt.Sprintf("search:%d", hash), searchTTL, func(ctx context.Context) (searchPage, error) {
		hits, hasMore, err := s.dbService.SearchText(ctx, filter)
		return searchPage{Hits: hits, HasMore: hasMore}, err
	})
	if err != nil {
		return nil, false, err
	}
	return page.Hits, page.HasMore, nil
}

// SearchImage is not cached, every upload is different
func (s *SearchService) SearchImage(ctx context.Context, filter *analogdb.SearchFilter) ([]analogdb.SearchHit, error) {
	return s.dbService.SearchImage(ctx, filter)
}

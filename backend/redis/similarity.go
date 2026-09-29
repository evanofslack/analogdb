package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/mitchellh/hashstructure/v2"

	"github.com/evanofslack/analogdb"
)

const (
	similarInstance  = "similar"
	similarLocalSize = 10000
	similarTTL       = time.Hour * 24
)

// ensure interface is implemented
var _ analogdb.SimilarityService = (*SimilarityService)(nil)

type SimilarityService struct {
	rdb          *RDB
	similarCache *Cache
	dbService    analogdb.SimilarityService
	postService  analogdb.PostService
}

func NewCacheSimilarityService(rdb *RDB, dbService analogdb.SimilarityService, postService analogdb.PostService) *SimilarityService {
	// stores the ordered ids of similar posts
	similarCache := rdb.NewCache(similarInstance, similarLocalSize, similarTTL)

	return &SimilarityService{
		rdb:          rdb,
		similarCache: similarCache,
		dbService:    dbService,
		postService:  postService,
	}
}

func (s *SimilarityService) CreateSchemas(ctx context.Context) error {
	return s.dbService.CreateSchemas(ctx)
}

func (s *SimilarityService) EncodePost(ctx context.Context, id int) error {
	return s.dbService.EncodePost(ctx, id)
}

func (s *SimilarityService) BatchEncodePosts(ctx context.Context, ids []int, batchSize int) ([]int, error) {
	return s.dbService.BatchEncodePosts(ctx, ids, batchSize)
}

// FindSimilarPosts loads the cached similar ids through the post service,
// so deleted posts drop out and patched posts show their current data.
func (s *SimilarityService) FindSimilarPosts(ctx context.Context, filter *analogdb.PostSimilarityFilter) ([]*analogdb.Post, error) {
	ids, err := s.FindSimilarPostIDs(ctx, filter)
	if err != nil {
		return nil, err
	}

	posts, _, err := s.postService.FindPosts(ctx, analogdb.NewPostFilterWithIDs(ids))
	if err != nil {
		return nil, err
	}
	return analogdb.OrderPostsByIDs(posts, ids), nil
}

func (s *SimilarityService) FindSimilarPostIDs(ctx context.Context, filter *analogdb.PostSimilarityFilter) ([]int, error) {
	if filter.ID == nil {
		return nil, fmt.Errorf("postID cannot be nil")
	}

	id := *filter.ID

	s.rdb.logger.DebugContext(ctx, "Start find similar post ids with cache", "instance", s.similarCache.instance, "post_id", id)
	defer s.rdb.logger.DebugContext(ctx, "Finish find similar post ids with cache", "instance", s.similarCache.instance, "post_id", id)

	// generate a unique hash from the filter struct
	hash, err := hashstructure.Hash(filter, hashstructure.FormatV2, nil)
	if err != nil {
		s.rdb.logger.ErrorContext(ctx, "Fail hash post similarity filter", "instance", s.similarCache.instance, "post_id", id, "error", err)

		// if we failed, fallback to db
		return s.dbService.FindSimilarPostIDs(ctx, filter)
	}

	key := fmt.Sprintf("similar:%d", hash)

	ids, err := fetch(ctx, s.similarCache, key, similarTTL, func(ctx context.Context) ([]int, error) {
		return s.dbService.FindSimilarPostIDs(ctx, filter)
	})
	if err != nil {
		return nil, err
	}
	return analogdb.DedupeIDs(ids), nil
}

func (s *SimilarityService) DeletePost(ctx context.Context, id int) error {
	return s.dbService.DeletePost(ctx, id)
}

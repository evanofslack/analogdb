package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/cache/v9"
	"github.com/mitchellh/hashstructure/v2"

	"github.com/evanofslack/analogdb"
)

const (
	similarInstance  = "similar"
	similarLocalSize = 10000
	similarTTL       = time.Hour * 24

	idKeysInstance  = "idkeys"
	idKeysLocalSize = 1000
	idKeysTTL       = time.Hour * 24

	delimiter = ";"
)

// ensure interface is implemented
var _ analogdb.SimilarityService = (*SimilarityService)(nil)

type SimilarityService struct {
	rdb          *RDB
	similarCache *Cache
	idKeysCache  *Cache
	dbService    analogdb.SimilarityService
}

func NewCacheSimilarityService(rdb *RDB, dbService analogdb.SimilarityService) *SimilarityService {
	// stores similar posts
	similarCache := rdb.NewCache(similarInstance, similarLocalSize, similarTTL)

	// map of post id to list of posts cache keys.
	// each post id can map to many similarity filters.
	// When a post is deleted, need to remove all matching
	// cached similarity filters.
	idKeysCache := rdb.NewCache(idKeysInstance, idKeysLocalSize, idKeysTTL)

	return &SimilarityService{
		rdb:          rdb,
		similarCache: similarCache,
		idKeysCache:  idKeysCache,
		dbService:    dbService,
	}
}

func (s *SimilarityService) CreateSchemas(ctx context.Context) error {
	return s.dbService.CreateSchemas(ctx)
}

func (s *SimilarityService) EncodePost(ctx context.Context, id int) error {
	return s.dbService.EncodePost(ctx, id)
}

func (s *SimilarityService) BatchEncodePosts(ctx context.Context, ids []int, batchSize int) error {
	return s.dbService.BatchEncodePosts(ctx, ids, batchSize)
}

func (s *SimilarityService) FindSimilarPosts(ctx context.Context, filter *analogdb.PostSimilarityFilter) ([]*analogdb.Post, error) {
	if filter.ID == nil {
		return nil, fmt.Errorf("postID cannot be nil")
	}

	id := *filter.ID

	s.rdb.logger.DebugContext(ctx, "Start find similar posts by image with cache", "instance", s.similarCache.instance, "post_id", id)
	defer func() {
		s.rdb.logger.DebugContext(ctx, "Finish find similar posts by image with cache", "instance", s.similarCache.instance, "post_id", id)
	}()

	// generate a unique hash from the filter struct
	hash, err := hashstructure.Hash(filter, hashstructure.FormatV2, nil)
	if err != nil {
		s.rdb.logger.ErrorContext(ctx, "Fail hash post similarity filter", "instance", s.similarCache.instance, "post_id", id, "error", err)

		// if we failed, fallback to db
		return s.dbService.FindSimilarPosts(ctx, filter)
	}

	postKey := fmt.Sprint(hash)
	idKey := fmt.Sprint(id)

	var posts []*analogdb.Post

	// try to get posts from the cache
	err = s.similarCache.get(ctx, postKey, &posts)

	// no error means we found in cache
	if err == nil {
		return posts, nil
	}

	// fallback to db
	posts, err = s.dbService.FindSimilarPosts(ctx, filter)
	if err != nil {
		return nil, err
	}

	// add similar posts to cache
	// do this async so response is returned quicker
	go func() {
		s.rdb.logger.DebugContext(ctx, "Added similar posts to cache", "instance", s.similarCache.instance, "post_id", id)
		// create a new context; orignal one will be canceled when request is closed
		ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
		defer cancel()

		s.similarCache.set(ctx, &cache.Item{
			Ctx:   ctx,
			Key:   postKey,
			Value: &posts,
			TTL:   similarTTL,
		})
	}()

	// update id's hashes in cache
	// do this async so response is returned quicker
	go func() {
		s.rdb.logger.DebugContext(ctx, "Added posts id hashes to cache", "instance", s.idKeysCache.instance, "post_id", id)
		// create a new context; orignal one will be canceled when request is closed
		ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
		defer cancel()

		// list of hashes seperated with delimiter
		var idKeyHashesString string

		// try to get id key's hashes from cache
		err = s.idKeysCache.get(ctx, idKey, &idKeyHashesString)

		// split string to list
		var idKeyHashes []string

		if idKeyHashesString != "" {
			idKeyHashes = strings.Split(idKeyHashesString, delimiter)
		}

		// if key already exists in slice, no more to do
		for _, h := range idKeyHashes {
			if postKey == h {
				s.rdb.logger.DebugContext(ctx, "Post key hash already exists in cache, skipping", "instance", s.idKeysCache.instance, "post_id", id, "hash", postKey)
				return
			}
		}

		// otherwise add it and save back to cache
		idKeyHashes = append(idKeyHashes, postKey)

		// serialize as string
		idKeyHashesString = strings.Join(idKeyHashes, delimiter)

		// save this back to the cache
		s.idKeysCache.set(ctx, &cache.Item{
			Ctx:   ctx,
			Key:   idKey,
			Value: &idKeyHashes,
			TTL:   similarTTL,
		})
		s.rdb.logger.DebugContext(ctx, "Added hash for post id", "instance", s.idKeysCache.instance, "post_id", id, "hash", postKey, "hashes", len(idKeyHashes))
	}()

	return posts, nil
}

func (s *SimilarityService) DeletePost(ctx context.Context, id int) error {
	s.rdb.logger.DebugContext(ctx, "Start delete vector post with cache", "post_id", id)
	defer s.rdb.logger.DebugContext(ctx, "Finish delete vector post with cache", "post_id", id)

	// remove from cache in background
	go func() {
		idKey := fmt.Sprint(id)

		// set of all hashes this id maps to
		var idKeyHashesString string

		// try to get id key's hashes from cache
		err := s.idKeysCache.get(ctx, idKey, &idKeyHashesString)
		if err != nil {
			return
		}

		// split string to list
		var idKeyHashes []string

		if idKeyHashesString != "" {
			idKeyHashes = strings.Split(idKeyHashesString, delimiter)
		}

		// for all hashes, remove from posts cache
		for _, hash := range idKeyHashes {
			s.rdb.logger.DebugContext(ctx, "Delete hash from similar posts cache", "post_id", id, "hash", hash, "instance", s.similarCache)
			s.similarCache.delete(ctx, hash)
		}

		// and remove from key ids cache
		s.rdb.logger.DebugContext(ctx, "Delete key from post ids cache", "post_id", id, "instance", s.idKeysCache)
		s.idKeysCache.delete(ctx, idKey)
	}()

	return s.dbService.DeletePost(ctx, id)
}

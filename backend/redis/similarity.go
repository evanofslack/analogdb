package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/cache/v8"
	"github.com/mitchellh/hashstructure/v2"

	"github.com/evanofslack/analogdb"
)

const (
	similarTTL       = time.Hour * 24
	similarLocalSize = 10000
)

// ensure interface is implemented
var _ analogdb.SimilarityService = (*SimilarityService)(nil)

type SimilarityService struct {
	rdb         *RDB
	postsCache  *cache.Cache
	idKeysCache *cache.Cache
	dbService   analogdb.SimilarityService
}

func NewCacheSimilarityService(rdb *RDB, dbService analogdb.SimilarityService) *SimilarityService {

	// stores similar posts
	postsCache := cache.New(&cache.Options{
		Redis:      rdb.db,
		LocalCache: cache.NewTinyLFU(1000, similarTTL),
	})

	// map of post id to list of posts cache keys.
	// each post id can map to many similarity filters.
	// When a post is deleted, need to remove all matching
	// cached similarity filters.
	idKeysCache := cache.New(&cache.Options{
		Redis:      rdb.db,
		LocalCache: cache.NewTinyLFU(1000, similarTTL),
	})

	return &SimilarityService{
		rdb:         rdb,
		postsCache:  postsCache,
		idKeysCache: idKeysCache,
		dbService:   dbService,
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

	s.rdb.logger.Debug().Int("postID", id).Msg("Starting find similar posts by image with cache")
	defer func() {
		s.rdb.logger.Debug().Int("postID", id).Msg("Finished find similar posts by image with cache")
	}()

	// generate a unique hash from the filter struct
	hash, err := hashstructure.Hash(filter, hashstructure.FormatV2, nil)
	if err != nil {
		s.rdb.logger.Err(err).Int("postID", id).Msg("Failed to hash post similarity filter")

		// if we failed, fallback to db
		return s.dbService.FindSimilarPosts(ctx, filter)
	}

	postKey := fmt.Sprint(hash)
	idKey := fmt.Sprint(id)

	s.rdb.logger.Debug().Int("postID", id).Str("hash", postKey).Msg("Generated post key hash from similarity filter")

	var posts []*analogdb.Post

	// try to get posts from the cache
	err = s.postsCache.Get(ctx, postKey, &posts)

	// no error means we found in cache
	if err == nil {
		s.rdb.logger.Debug().Int("postID", id).Msg("Found similar posts in cache")
		return posts, nil
	}

	s.rdb.logger.Info().Int("postID", id).Str("error", err.Error()).Msg("Failed to find similar posts in cache")

	posts, err = s.dbService.FindSimilarPosts(ctx, filter)
	if err != nil {
		return nil, err
	}

	// add similar posts to cache
	// do this async so response is returned quicker
	go func() {

		s.rdb.logger.Debug().Msg("Adding similar posts to cache")
		// create a new context; orignal one will be canceled when request is closed
		ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
		defer cancel()

		if err := s.postsCache.Set(&cache.Item{
			Ctx:   ctx,
			Key:   postKey,
			Value: &posts,
			TTL:   similarTTL,
		}); err != nil {
			s.rdb.logger.Err(err).Msg("Failed to add similar posts to cache")
		} else {
			s.rdb.logger.Debug().Msg("Added similar posts to cache")
		}
	}()

	// update id's hashes in cache
	// do this async so response is returned quicker
	go func() {

		s.rdb.logger.Debug().Msg("Adding id hashes to cache")
		// create a new context; orignal one will be canceled when request is closed
		ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
		defer cancel()

		// slice of all hashes this id maps to
		// would rather use a map/set but it doesn't serialize correctly
		var idKeyHashes []string

		// try to get id key's hashes from cache
		err = s.idKeysCache.Get(ctx, idKey, &idKeyHashes)

		if err == nil {
			s.rdb.logger.Debug().Int("postID", id).Msg("Found id's hashes in cache")
		} else {
			s.rdb.logger.Info().Int("postID", id).Str("error", err.Error()).Msg("Failed to find id's hashes in cache")
		}

		// check if key already in slice; if not, add it.
		for _, h := range idKeyHashes {
			if postKey == h {
				s.rdb.logger.Debug().Int("postID", id).Str("hash", postKey).Msg("Post key hash already exists in cache, skipping")
				return
			}
		}

		idKeyHashes = append(idKeyHashes, postKey)

		s.rdb.logger.Debug().Int("postID", id).Str("hash", postKey).Msg(fmt.Sprintf("Added hash for postID, now has %d hashes", len(idKeyHashes)))

		// save this back to the cache
		if err := s.idKeysCache.Set(&cache.Item{
			Ctx:   ctx,
			Key:   idKey,
			Value: &idKeyHashes,
			TTL:   similarTTL,
		}); err != nil {
			s.rdb.logger.Err(err).Msg("Failed to add id hashes to cache")
		} else {
			s.rdb.logger.Debug().Msg("Added id hashes to cache")
		}
	}()

	return posts, nil
}

func (s *SimilarityService) DeletePost(ctx context.Context, id int) error {

	s.rdb.logger.Debug().Int("postID", id).Msg("Starting delete vector post with cache")
	defer func() {
		s.rdb.logger.Debug().Int("postID", id).Msg("Finished delete vector post with cache")
	}()

	// remove from cache in background
	go func() {
		idKey := fmt.Sprint(id)

		// set of all hashes this id maps to
		var idKeyHashes []string

		// try to get id key's hashes from cache
		err := s.idKeysCache.Get(ctx, idKey, &idKeyHashes)

		if err == nil {
			s.rdb.logger.Debug().Int("postID", id).Msg("Found id's hashes in cache")
		} else {
			s.rdb.logger.Info().Int("postID", id).Str("error", err.Error()).Msg("Failed to find id's hashes in cache")
			return
		}

		// for all hashes, remove from posts cache
		for _, hash := range idKeyHashes {
			s.rdb.logger.Debug().Int("postID", id).Str("hash", hash).Msg("Deleting hash from similar posts cache")
			if err := s.postsCache.Delete(ctx, hash); err != nil {
				s.rdb.logger.Info().Str("error", err.Error()).Int("postID", id).Str("hash", hash).Msg("Failed deleting hash from similar posts cache")
			} else {
				s.rdb.logger.Debug().Int("postID", id).Str("hash", hash).Msg("Success deleting hash from similar posts cache")
			}
		}

		// and remove from key ids cache
		s.rdb.logger.Debug().Int("postID", id).Msg("Deleting key from post ids cache")
		if err := s.idKeysCache.Delete(ctx, idKey); err != nil {
			s.rdb.logger.Info().Str("error", err.Error()).Int("postID", id).Msg("Failed deleting key from key ids cache")
		} else {
			s.rdb.logger.Info().Int("postID", id).Msg("Success deleting key from key ids cache")
		}
	}()

	return s.dbService.DeletePost(ctx, id)
}

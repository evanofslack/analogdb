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
	// timeout for cache operations
	cacheOpTimeout = time.Second * 5

	// name of post cache
	postInstance = "post"
	// ttl for individual post in cache
	postTTL = time.Hour * 24
	// im memory cache size for individual posts
	postLocalSize = 1000

	// name of posts cache
	postsInstance = "posts"
	// ttl for all other post service data
	postsTTL = time.Hour * 4
	// in memory cache size all other post service data
	postsLocalSize = 100
)

// ensure interface is implemented
var _ analogdb.PostService = (*PostService)(nil)

type PostService struct {
	rdb        *RDB
	postCache  *Cache
	postsCache *Cache
	dbService  analogdb.PostService
}

func NewCachePostService(rdb *RDB, dbService analogdb.PostService) *PostService {

	postCache := rdb.NewCache(postInstance, postLocalSize, postTTL)
	postsCache := rdb.NewCache(postsInstance, postsLocalSize, postsTTL)

	return &PostService{
		rdb:        rdb,
		postCache:  postCache,
		postsCache: postsCache,
		dbService:  dbService,
	}
}

func (s *PostService) CreatePost(ctx context.Context, post *analogdb.CreatePost) (*analogdb.Post, error) {
	return s.dbService.CreatePost(ctx, post)
}

func (s *PostService) FindPosts(ctx context.Context, filter *analogdb.PostFilter) ([]*analogdb.Post, int, error) {

	s.rdb.logger.Debug().Msg("Starting find posts with cache")
	defer func() {
		s.rdb.logger.Debug().Msg("Finished find posts with cache")
	}()

	// generate a unique hash from the filter struct
	hash, err := hashstructure.Hash(filter, hashstructure.FormatV2, nil)
	if err != nil {
		s.rdb.logger.Err(err).Msg("Failed to hash post filter")

		// if we failed, fallback to db
		return s.dbService.FindPosts(ctx, filter)
	}

	postsHash := fmt.Sprint(hash)
	postsCountHash := fmt.Sprintf("%s-%s", postsHash, "count")

	var posts []*analogdb.Post
	var count int

	// try to get posts from the cache
	err = s.postsCache.cache.Get(ctx, postsHash, &posts)
	if err != nil {
		s.rdb.logger.Info().Str("error", err.Error()).Msg("Failed to find posts from cache")
	}
	// try to get posts count from the cache
	err = s.postsCache.cache.Get(ctx, postsCountHash, &count)

	// no error means we found in cache
	if err == nil {
		s.rdb.logger.Debug().Msg("Found posts and posts count in cache")
		return posts, count, nil
	}

	// otherwise we must fallback to db
	s.rdb.logger.Info().Str("error", err.Error()).Msg("Failed to find posts count from cache")
	posts, count, err = s.dbService.FindPosts(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// add posts to cache
	// do this async so response is returned quicker
	go func() {

		s.rdb.logger.Debug().Msg("Adding posts and posts counts to cache")
		// create a new context; orignal one will be canceled when request is closed
		ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
		defer cancel()

		if err := s.postsCache.cache.Set(&cache.Item{
			Ctx:   ctx,
			Key:   postsHash,
			Value: &posts,
			TTL:   postsTTL,
		}); err != nil {
			s.rdb.logger.Err(err).Msg("Failed to add posts to cache")
		} else {
			s.rdb.logger.Debug().Msg("Added posts to cache")
		}
		// add posts count to cache
		if err := s.postsCache.cache.Set(&cache.Item{
			Ctx:   ctx,
			Key:   postsCountHash,
			Value: &count,
			TTL:   postsTTL,
		}); err != nil {
			s.rdb.logger.Err(err).Msg("Failed to add posts count to cache")
		} else {
			s.rdb.logger.Debug().Msg("Added posts count to cache")
		}
	}()

	return posts, count, nil
}

func (s *PostService) FindPostByID(ctx context.Context, id int) (*analogdb.Post, error) {

	s.rdb.logger.Debug().Int("postID", id).Msg("Starting find post by id with cache")
	defer func() {
		s.rdb.logger.Debug().Int("postID", id).Msg("Finished find post by id with cache")
	}()

	var post *analogdb.Post
	postKey := fmt.Sprint(id)

	// try to get post from the cache
	err := s.postCache.cache.Get(ctx, postKey, &post)

	// no error means we found in cache
	if err == nil {
		s.rdb.logger.Debug().Int("postID", id).Msg("Found post by id in cache")
		return post, nil
	}

	// otherwise we must fallback to db
	s.rdb.logger.Info().Int("postID", id).Str("error", err.Error()).Msg("Failed to find post by id from cache")
	post, err = s.dbService.FindPostByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// add post to cache
	// do this async so response is returned quicker
	go func() {

		s.rdb.logger.Debug().Int("postID", id).Msg("Adding post to cache")
		// create a new context; orignal one will be canceled when request is closed
		ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
		defer cancel()

		// add to cache
		if err := s.postCache.cache.Set(&cache.Item{
			Ctx:   ctx,
			Key:   postKey,
			Value: &post,
			TTL:   postTTL,
		}); err != nil {
			s.rdb.logger.Err(err).Int("postID", id).Msg("Failed to add post to cache")
		} else {
			s.rdb.logger.Debug().Int("postID", id).Msg("Added post to cache")
		}
	}()
	return post, nil
}

func (s *PostService) PatchPost(ctx context.Context, patch *analogdb.PatchPost, id int) error {

	s.rdb.logger.Debug().Int("postID", id).Msg("Starting patch post with cache")
	defer func() {
		s.rdb.logger.Debug().Int("postID", id).Msg("Finished patch post with cache")
	}()

	// remove post from the cache
	go s.removePostFromCache(id)

	return s.dbService.PatchPost(ctx, patch, id)
}

func (s *PostService) DeletePost(ctx context.Context, id int) error {

	s.rdb.logger.Debug().Int("postID", id).Msg("Starting delete post with cache")
	defer func() {
		s.rdb.logger.Debug().Int("postID", id).Msg("Finished delete post with cache")
	}()

	// cache is now stale, delete old entries
	go func() {
		s.removePostFromCache(id)
	}()

	return s.dbService.DeletePost(ctx, id)
}

func (s *PostService) AllPostIDs(ctx context.Context) ([]int, error) {
	return s.dbService.AllPostIDs(ctx)
}

func (s *PostService) removePostFromCache(id int) {

	s.rdb.logger.Debug().Int("postID", id).Msg("Removing post from cache")

	postKey := fmt.Sprint(id)

	// create a new context
	ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
	defer cancel()
	if err := s.postCache.cache.Delete(ctx, postKey); err != nil {
		s.rdb.logger.Info().Str("error", err.Error()).Int("postID", id).Msg("Failed to remove post from cache")
	} else {
		s.rdb.logger.Debug().Int("postID", id).Msg("Removed post from cache")
	}
}

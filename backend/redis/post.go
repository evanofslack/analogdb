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

	s.rdb.logger.Debug().Str("instance", s.postsCache.instance).Msg("Starting find posts with cache")
	defer func() {
		s.rdb.logger.Debug().Str("instance", s.postsCache.instance).Msg("Finished find posts with cache")
	}()

	// generate a unique hash from the filter struct
	hash, err := hashstructure.Hash(filter, hashstructure.FormatV2, nil)
	if err != nil {
		s.rdb.logger.Err(err).Str("instance", s.postsCache.instance).Msg("Failed to hash post filter")

		// if we failed, fallback to db
		return s.dbService.FindPosts(ctx, filter)
	}

	postsHash := fmt.Sprint(hash)
	postsCountHash := fmt.Sprintf("%s-%s", postsHash, "count")

	var posts []*analogdb.Post
	var count int

	// try to get posts from cache
	err = s.postCache.get(ctx, postsHash, &posts)

	// try to get posts count from cache
	err = s.postsCache.get(ctx, postsCountHash, &count)

	// no error means we found in cache
	if err == nil {
		return posts, count, nil
	}

	// fallback to db
	posts, count, err = s.dbService.FindPosts(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// add posts to cache
	// do this async so response is returned quicker
	go func() {

		s.rdb.logger.Debug().Str("instance", s.postsCache.instance).Msg("Adding posts and posts counts to cache")
		// create a new context; orignal one will be canceled when request is closed
		ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
		defer cancel()

		// add posts to cache
		s.postsCache.set(&cache.Item{
			Ctx:   ctx,
			Key:   postsHash,
			Value: &posts,
			TTL:   postsTTL,
		})
		// add posts count to cache
		s.postsCache.set(&cache.Item{
			Ctx:   ctx,
			Key:   postsCountHash,
			Value: &count,
			TTL:   postsTTL,
		})
	}()

	return posts, count, nil
}

func (s *PostService) FindPostByID(ctx context.Context, id int) (*analogdb.Post, error) {

	s.rdb.logger.Debug().Str("instance", s.postCache.instance).Int("postID", id).Msg("Starting find post by id with cache")
	defer func() {
		s.rdb.logger.Debug().Str("instance", s.postCache.instance).Int("postID", id).Msg("Finished find post by id with cache")
	}()

	var post *analogdb.Post
	postKey := fmt.Sprint(id)

	// try to get post from the cache
	err := s.postCache.get(ctx, postKey, &post)

	// no error means we found in cache
	if err == nil {
		return post, nil
	}

	// error means we must fallback to db
	post, err = s.dbService.FindPostByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// add post to cache
	// do this async so response is returned quicker
	go func() {

		s.rdb.logger.Debug().Str("instance", s.postCache.instance).Int("postID", id).Msg("Adding post to cache")
		// create a new context; orignal one will be canceled when request is closed
		ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
		defer cancel()

		// add to cache
		s.postCache.set(&cache.Item{
			Ctx:   ctx,
			Key:   postKey,
			Value: &post,
			TTL:   postTTL,
		})
	}()
	return post, nil
}

func (s *PostService) PatchPost(ctx context.Context, patch *analogdb.PatchPost, id int) error {

	s.rdb.logger.Debug().Str("instance", s.postCache.instance).Int("postID", id).Msg("Starting patch post with cache")
	defer func() {
		s.rdb.logger.Debug().Str("instance", s.postCache.instance).Int("postID", id).Msg("Finished patch post with cache")
	}()

	// remove post from the cache
	go s.removePostFromCache(id)

	return s.dbService.PatchPost(ctx, patch, id)
}

func (s *PostService) DeletePost(ctx context.Context, id int) error {

	s.rdb.logger.Debug().Str("instance", s.postCache.instance).Int("postID", id).Msg("Starting delete post with cache")
	defer func() {
		s.rdb.logger.Debug().Str("instance", s.postCache.instance).Int("postID", id).Msg("Finished delete post with cache")
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

	s.rdb.logger.Debug().Str("instance", s.postCache.instance).Int("postID", id).Msg("Removing post from cache")

	postKey := fmt.Sprint(id)

	// create a new context
	ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
	defer cancel()
	s.postCache.delete(ctx, postKey)
}

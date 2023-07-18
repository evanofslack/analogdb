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

	// ttl for individual posts in cache
	postTTL = time.Hour * 24
	// im memory cache size for individual posts
	postLocalSize = 1000

	// ttl for all other post service data
	generalTTL = time.Hour * 4
	// in memory cache size all other post service data
	generalLocalSize = 100

	// key for all post ids
	allPostIDsKey = "allpostids"
)

// ensure interface is implemented
var _ analogdb.PostService = (*PostService)(nil)

type PostService struct {
	rdb       *RDB
	postCache *cache.Cache
	genCache  *cache.Cache
	dbService analogdb.PostService
}

func NewCachePostService(rdb *RDB, dbService analogdb.PostService) *PostService {

	postCache := cache.New(&cache.Options{
		Redis:      rdb.db,
		LocalCache: cache.NewTinyLFU(postLocalSize, postTTL),
	})

	genCache := cache.New(&cache.Options{
		Redis:      rdb.db,
		LocalCache: cache.NewTinyLFU(generalLocalSize, generalTTL),
	})

	return &PostService{
		rdb:       rdb,
		postCache: postCache,
		genCache:  genCache,
		dbService: dbService,
	}
}

func (s *PostService) CreatePost(ctx context.Context, post *analogdb.CreatePost) (*analogdb.Post, error) {

	s.rdb.logger.Debug().Msg("Starting create post with cache")
	defer func() {
		s.rdb.logger.Debug().Msg("Finished create post with cache")
	}()

	// cache is now stale, remove old entries
	go s.removeAllPostIDsFromCache()
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
	err = s.genCache.Get(ctx, postsHash, &posts)
	if err != nil {
		s.rdb.logger.Info().Str("error", err.Error()).Msg("Failed to find posts from cache")
	}
	// try to get posts count from the cache
	err = s.genCache.Get(ctx, postsCountHash, &count)

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

		if err := s.genCache.Set(&cache.Item{
			Ctx:   ctx,
			Key:   postsHash,
			Value: &posts,
			TTL:   generalTTL,
		}); err != nil {
			s.rdb.logger.Err(err).Msg("Failed to add posts to cache")
		} else {
			s.rdb.logger.Debug().Msg("Added posts to cache")
		}
		// add posts count to cache
		if err := s.genCache.Set(&cache.Item{
			Ctx:   ctx,
			Key:   postsCountHash,
			Value: &count,
			TTL:   generalTTL,
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
	err := s.postCache.Get(ctx, postKey, &post)

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
		if err := s.postCache.Set(&cache.Item{
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
		s.removeAllPostIDsFromCache()
	}()

	return s.dbService.DeletePost(ctx, id)
}

func (s *PostService) AllPostIDs(ctx context.Context) ([]int, error) {

	s.rdb.logger.Debug().Msg("Starting get all post ids with cache")
	defer func() {
		s.rdb.logger.Debug().Msg("Finished get all post ids with cache")
	}()

	var ids []int

	// try to get from the cache
	err := s.genCache.Get(ctx, allPostIDsKey, &ids)
	if err == nil {
		s.rdb.logger.Debug().Msg("Found all post ids in cache")
		return ids, nil
	}

	// fallback to postgres if not in cache
	s.rdb.logger.Info().Str("error", err.Error()).Msg("Failed to get all post ids from cache")
	ids, err = s.dbService.AllPostIDs(ctx)
	if err != nil {
		return nil, err
	}

	// add to cache
	// do this async so response is returned quicker
	go func() {

		s.rdb.logger.Debug().Msg("Adding all post ids to cache")
		// create a new context; orignal one will be canceled when request is closed
		ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
		defer cancel()

		if err := s.genCache.Set(&cache.Item{
			Ctx:   ctx,
			Key:   allPostIDsKey,
			Value: &ids,
			TTL:   generalTTL,
		}); err != nil {
			s.rdb.logger.Err(err).Msg("Failed to add all post ids to cache")
		} else {
			s.rdb.logger.Debug().Msg("Added all post ids to cache")
		}
	}()
	return ids, nil
}

func (s *PostService) removePostFromCache(id int) {

	s.rdb.logger.Debug().Int("postID", id).Msg("Removing post from cache")

	postKey := fmt.Sprint(id)

	// create a new context
	ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
	defer cancel()
	if err := s.postCache.Delete(ctx, postKey); err != nil {
		s.rdb.logger.Info().Str("error", err.Error()).Int("postID", id).Msg("Failed to remove post from cache")
	} else {
		s.rdb.logger.Debug().Int("postID", id).Msg("Removed post from cache")
	}
}

func (s *PostService) removeAllPostIDsFromCache() {

	s.rdb.logger.Debug().Msg("Removing all post ids from cache")

	// create a new context
	ctx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
	defer cancel()
	if err := s.genCache.Delete(ctx, allPostIDsKey); err != nil {
		s.rdb.logger.Info().Str("error", err.Error()).Msg("Failed to remove all post ids from cache")
	} else {
		s.rdb.logger.Debug().Msg("Removed all post ids from cache")
	}
}

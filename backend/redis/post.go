package redis

import (
	"context"
	"fmt"
	"time"
	goTime "time"

	"github.com/go-redis/cache/v8"
	"github.com/mitchellh/hashstructure/v2"

	"github.com/evanofslack/analogdb"
)

const (
	postsTTL        = goTime.Hour * 4
	postTTL        = goTime.Hour * 24
	cacheSetTimeout = time.Second * 5
)

// ensure interface is implemented
var _ analogdb.PostService = (*PostService)(nil)

type PostService struct {
	rdb       *RDB
	cache     *cache.Cache
	dbService analogdb.PostService
}

func NewCachePostService(rdb *RDB, dbService analogdb.PostService) *PostService {
	cache := cache.New(&cache.Options{
		Redis:      rdb.db,
		LocalCache: cache.NewTinyLFU(1000, authorsTTL),
	})

	return &PostService{
		rdb:       rdb,
		cache:     cache,
		dbService: dbService,
	}
}

// This is a passthrough to the database
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
		s.rdb.logger.Info().Str("error", err.Error()).Msg("Failed to hash post filter")

		// if we failed, fallback to db
		return s.dbService.FindPosts(ctx, filter)
	}

	postsHash := fmt.Sprint(hash)
	postsCountHash := fmt.Sprintf("%s-%s", postsHash, "count")

	var posts []*analogdb.Post
	var count int

	// try to get posts from the cache
	err = s.cache.Get(ctx, string(postsHash), &posts)
	if err != nil {
		s.rdb.logger.Info().Str("error", err.Error()).Msg("Failed to find posts from cache")
	}
	// try to get posts count from the cache
	err = s.cache.Get(ctx, string(postsCountHash), &count)

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
		ctx, cancel := context.WithTimeout(context.Background(), cacheSetTimeout)
		defer cancel()

		if err := s.cache.Set(&cache.Item{
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
		if err := s.cache.Set(&cache.Item{
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
	err := s.cache.Get(ctx, postKey, &post)
	if err != nil {
		s.rdb.logger.Info().Int("postID", id).Str("error", err.Error()).Msg("Failed to find post by id from cache")
	}

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
		ctx, cancel := context.WithTimeout(context.Background(), cacheSetTimeout)
		defer cancel()

		// add to cache
		if err := s.cache.Set(&cache.Item{
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
	return s.PatchPost(ctx, patch, id)
}

func (s *PostService) DeletePost(ctx context.Context, id int) error {
	return s.DeletePost(ctx, id)
}

func (s *PostService) AllPostIDs(ctx context.Context) ([]int, error) {
	return s.AllPostIDs(ctx)
}

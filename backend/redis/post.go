package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/mitchellh/hashstructure/v2"

	"github.com/evanofslack/analogdb"
)

const (
	// timeout for cache operations
	cacheOpTimeout = time.Second * 5
	// timeout for removing a post after a write
	evictTimeout = time.Second

	// name of post cache
	postInstance = "post"
	// ttl for individual post in cache
	postTTL = time.Hour * 24
	// how many posts one delete removes when every post is invalidated
	invalidateBatch = 1000
	// im memory cache size for individual posts
	postLocalSize = 1000

	// name of posts cache
	postsInstance = "posts"
	// ttl for all other post service data
	postsTTL = time.Hour * 1
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
	created, err := s.dbService.CreatePost(ctx, post)
	if err != nil {
		return nil, err
	}
	s.rdb.bumpGen(ctx, postsEntity, filmsEntity, camerasEntity)
	return created, nil
}

func (s *PostService) FindPosts(ctx context.Context, filter *analogdb.PostFilter) ([]*analogdb.Post, int, error) {
	s.rdb.logger.DebugContext(ctx, "Start find posts with cache", "instance", s.postsCache.instance)
	defer s.rdb.logger.DebugContext(ctx, "Finish find posts with cache", "instance", s.postsCache.instance)

	// generate a unique hash from the filter struct
	hash, err := hashstructure.Hash(filter, hashstructure.FormatV2, nil)
	if err != nil {
		s.rdb.logger.ErrorContext(ctx, "Fail hash post filter", "instance", s.postsCache.instance, "error", err)

		// if we failed, fallback to db
		return s.dbService.FindPosts(ctx, filter)
	}

	// the count does not depend on the page or order, so all pages share it
	countFilter := *filter
	countFilter.Limit = nil
	countFilter.Cursor = nil
	countFilter.Sort = nil
	if filter.IsRainbow() {
		// the rainbow only counts posts with a representative color
		countFilter.Sort = filter.Sort
	}
	countFilter.Order = nil
	countFilter.Seed = nil
	countHash, err := hashstructure.Hash(countFilter, hashstructure.FormatV2, nil)
	if err != nil {
		s.rdb.logger.ErrorContext(ctx, "Fail hash post count filter", "instance", s.postsCache.instance, "error", err)
		return s.dbService.FindPosts(ctx, filter)
	}

	postsKey := s.rdb.genCacheKey(ctx, postsEntity, fmt.Sprint(hash))
	countKey := s.rdb.genCacheKey(ctx, postsEntity, fmt.Sprintf("count:%d", countHash))

	var posts []*analogdb.Post
	var count int

	// try to get posts and count from cache
	postsErr := s.postsCache.get(ctx, postsKey, &posts)
	countErr := s.postsCache.get(ctx, countKey, &count)

	// no error means we found in cache
	if postsErr == nil && countErr == nil {
		return posts, count, nil
	}

	// fallback to db
	v, err := s.postsCache.once(ctx, postsKey, func(ctx context.Context) (any, error) {
		// a load that finished after our miss may have stored them already
		var cached postsResult
		if s.postsCache.lookup(ctx, postsKey, &cached.posts) == nil && s.postsCache.lookup(ctx, countKey, &cached.count) == nil {
			return cached, nil
		}
		posts, count, err := s.dbService.FindPosts(ctx, filter)
		if err != nil {
			return nil, err
		}
		s.postsCache.set(ctx, postsKey, posts, postsTTL)
		s.postsCache.set(ctx, countKey, count, postsTTL)
		return postsResult{posts: posts, count: count}, nil
	})
	if err != nil {
		return nil, 0, err
	}
	result := v.(postsResult)
	return result.posts, result.count, nil
}

type postsResult struct {
	posts []*analogdb.Post
	count int
}

func (s *PostService) FindPostByID(ctx context.Context, id int) (*analogdb.Post, error) {
	s.rdb.logger.DebugContext(ctx, "Start find post by id with cache", "instance", s.postCache.instance, "post_id", id)
	defer s.rdb.logger.DebugContext(ctx, "Finish find post by id with cache", "instance", s.postCache.instance, "post_id", id)

	var post *analogdb.Post
	postKey := fmt.Sprint(id)

	// try to get post from the cache
	err := s.postCache.get(ctx, postKey, &post)

	// no error means we found in cache
	if err == nil {
		return post, nil
	}

	// error means we must fallback to db
	v, err := s.postCache.once(ctx, postKey, func(ctx context.Context) (any, error) {
		// a load that finished after our miss may have stored it already
		var cached *analogdb.Post
		if s.postCache.lookup(ctx, postKey, &cached) == nil {
			return cached, nil
		}

		before, genErr := s.rdb.readGen(ctx, postsEntity)
		post, err := s.dbService.FindPostByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if genErr != nil {
			return post, nil
		}

		// a write during the lookup may have made this post stale. Check after the set, a write
		// either bumped the generation before this check or deletes the key after our set
		s.postCache.set(ctx, postKey, post, postTTL)
		if after, err := s.rdb.readGen(ctx, postsEntity); err != nil || after != before {
			_ = s.postCache.delete(ctx, postKey)
		}
		return post, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*analogdb.Post), nil
}

func (s *PostService) PatchPost(ctx context.Context, patch *analogdb.PatchPost, id int) error {
	s.rdb.logger.DebugContext(ctx, "Start patch post with cache", "instance", s.postCache.instance, "post_id", id)
	defer s.rdb.logger.DebugContext(ctx, "Finish patch post with cache", "instance", s.postCache.instance, "post_id", id)

	s.removePostFromCache(ctx, id)
	if err := s.dbService.PatchPost(ctx, patch, id); err != nil {
		return err
	}
	s.invalidatePost(ctx, id)
	return nil
}

func (s *PostService) DeletePost(ctx context.Context, id int, reason string) error {
	s.rdb.logger.DebugContext(ctx, "Start delete post with cache", "instance", s.postCache.instance, "post_id", id)
	defer s.rdb.logger.DebugContext(ctx, "Finish delete post with cache", "instance", s.postCache.instance, "post_id", id)

	s.removePostFromCache(ctx, id)
	if err := s.dbService.DeletePost(ctx, id, reason); err != nil {
		return err
	}
	s.invalidatePost(ctx, id)
	return nil
}

func (s *PostService) AllPostIDs(ctx context.Context) ([]int, error) {
	return s.dbService.AllPostIDs(ctx)
}

// invalidatePost runs after a write is committed
func (s *PostService) invalidatePost(ctx context.Context, id int) {
	s.rdb.bumpGen(ctx, postsEntity, filmsEntity, camerasEntity)
	s.removePostFromCache(ctx, id)
	s.rdb.stats.invalidations.WithLabelValues(postInstance).Inc()
}

// invalidateAll runs after a write to every post is committed
func (s *PostService) invalidateAll(ctx context.Context) error {
	ids, err := s.dbService.AllPostIDs(ctx)
	if err != nil {
		return err
	}
	s.rdb.bumpGen(ctx, postsEntity, filmsEntity, camerasEntity)
	for start := 0; start < len(ids); start += invalidateBatch {
		keys := []string{}
		for _, id := range ids[start:min(start+invalidateBatch, len(ids))] {
			keys = append(keys, fmt.Sprint(id))
		}
		if err := s.postCache.deleteMany(ctx, keys); err != nil {
			return err
		}
	}
	s.rdb.stats.invalidations.WithLabelValues(postInstance).Inc()
	return nil
}

func (s *PostService) removePostFromCache(ctx context.Context, id int) {
	s.rdb.logger.DebugContext(ctx, "Remove post from cache", "instance", s.postCache.instance, "post_id", id)

	postKey := fmt.Sprint(id)

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), evictTimeout)
	defer cancel()
	if err := s.postCache.delete(ctx, postKey); err != nil {
		s.rdb.logger.ErrorContext(ctx, "Fail remove post from cache", "instance", s.postCache.instance, "post_id", id, "error", err)
	}
}

package server

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/evanofslack/analogdb"
	"golang.org/x/sync/singleflight"
)

const missingVectorsMaxAge = time.Minute

// missingVectorCache holds the last diff of posts against vector objects
type missingVectorCache struct {
	mu      sync.Mutex
	missing []int
	extra   int
	fetched time.Time
	group   singleflight.Group
}

type missingVectorsResult struct {
	missing []int
	extra   int
}

var errVectorsUnavailable = &analogdb.Error{Code: analogdb.ERRUNAVAILABLE, Message: "vector database unavailable"}

// missingVectors returns the ids of posts with no vector object, newest first,
// and how many vector objects belong to posts that no longer exist
func (s *Server) missingVectors(ctx context.Context) ([]int, int, error) {
	c := &s.missingVectorCache
	c.mu.Lock()
	if c.missing != nil && time.Since(c.fetched) < missingVectorsMaxAge {
		missing, extra := c.missing, c.extra
		c.mu.Unlock()
		return missing, extra, nil
	}
	c.mu.Unlock()

	if s.VectorLister == nil {
		return nil, 0, errVectorsUnavailable
	}

	v, err, _ := c.group.Do("vectors", func() (any, error) {
		ctx := context.WithoutCancel(ctx)
		vectorIDs, err := s.VectorLister.VectorPostIDs(ctx)
		if err != nil {
			s.logger.ErrorContext(ctx, "Fail list vector post ids", "error", err)
			return nil, errVectorsUnavailable
		}
		postIDs, err := s.PostService.AllPostIDs(ctx)
		if err != nil {
			return nil, err
		}
		missing, extra := diffVectorIDs(postIDs, vectorIDs)
		c.mu.Lock()
		c.missing, c.extra, c.fetched = missing, extra, time.Now()
		c.mu.Unlock()
		return missingVectorsResult{missing: missing, extra: extra}, nil
	})
	if err != nil {
		return nil, 0, err
	}
	res := v.(missingVectorsResult)
	return res.missing, res.extra, nil
}

func diffVectorIDs(postIDs, vectorIDs []int) ([]int, int) {
	vectors := make(map[int]struct{}, len(vectorIDs))
	for _, id := range vectorIDs {
		vectors[id] = struct{}{}
	}
	posts := make(map[int]struct{}, len(postIDs))
	missing := []int{}
	for _, id := range postIDs {
		posts[id] = struct{}{}
		if _, ok := vectors[id]; !ok {
			missing = append(missing, id)
		}
	}
	extra := 0
	for id := range vectors {
		if _, ok := posts[id]; !ok {
			extra++
		}
	}
	slices.Sort(missing)
	slices.Reverse(missing)
	return slices.Compact(missing), extra
}

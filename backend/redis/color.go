package redis

import (
	"context"

	"github.com/evanofslack/analogdb"
)

// ensure interface is implemented
var _ analogdb.ColorService = (*ColorService)(nil)

type ColorService struct {
	dbService analogdb.ColorService
	posts     *PostService
}

func NewCacheColorService(dbService analogdb.ColorService, posts *PostService) *ColorService {
	return &ColorService{dbService: dbService, posts: posts}
}

// RederiveColors can change every post, so it drops every cached post and post
// list, also after a failure since earlier batches are committed
func (s *ColorService) RederiveColors(ctx context.Context) (*analogdb.ColorRederive, error) {
	result, err := s.dbService.RederiveColors(ctx)
	if invalidateErr := s.posts.invalidateAll(context.WithoutCancel(ctx)); invalidateErr != nil {
		s.posts.rdb.logger.ErrorContext(ctx, "Fail invalidate posts after color rederive", "error", invalidateErr)
		if err == nil {
			err = invalidateErr
		}
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

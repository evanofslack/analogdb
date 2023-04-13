package weaviate

import (
	"context"

	"github.com/evanofslack/analogdb"
)

var _ analogdb.SimilarityService = (*SimilarityService)(nil)

type SimilarityService struct {
	db          *DB
	postService analogdb.PostService
}

func NewSimilarityService(db *DB, ps analogdb.PostService) *SimilarityService {
	return &SimilarityService{db: db, postService: ps}
}

func (ss SimilarityService) FindSimilarPostsByImage(ctx context.Context, id int) ([]*analogdb.Post, error) {
	return []*analogdb.Post{}, nil
}

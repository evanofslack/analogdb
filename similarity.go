package analogdb

import "context"

type PostSimilarity struct {
	Post  Post    `json:"post"`
	Score float64 `json:"score"`
}

type SimilarityService interface {
	CreateSchemas(ctx context.Context) error
	BatchEncodePosts(ctx context.Context, ids []int) error
	FindSimilarPostsByImage(ctx context.Context, id int) ([]*Post, error)
}

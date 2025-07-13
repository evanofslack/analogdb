package analogdb

import "context"

type PostSimilarity struct {
    Post  Post    `json:"post"`
    Score float64 `json:"score" example:"0.73"`
}

type SimilarityService interface {
	CreateSchemas(ctx context.Context) error
	EncodePost(ctx context.Context, id int) error
	BatchEncodePosts(ctx context.Context, ids []int, batchSize int) error
	FindSimilarPosts(ctx context.Context, filter *PostSimilarityFilter) ([]*Post, error)
	DeletePost(ctx context.Context, id int) error
}

// used to enable encoding in http request
// only used to bypass encoding when running tests
type ContextKey string

const EncodeContextKey ContextKey = "encode"

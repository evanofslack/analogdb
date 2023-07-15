package analogdb

import "context"

type PostSimilarity struct {
	Post  Post    `json:"post"`
	Score float64 `json:"score"`
}

type SimilarityService interface {
	CreateSchemas(ctx context.Context) error
	EncodePost(ctx context.Context, id int) error
	BatchEncodePosts(ctx context.Context, ids []int, batchSize int) error
	FindSimilarPostsByImage(ctx context.Context, id int, filter *PostSimilarityFilter) ([]*Post, error)
	DeletePost(ctx context.Context, id int) error
}

// used to enable encoding in http request
// only used to bypass encoding when running tests
type ContextKey string

const EncodeContextKey ContextKey = "encode"

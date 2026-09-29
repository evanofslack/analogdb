package analogdb

import "context"

type PostSimilarity struct {
    Post  Post    `json:"post"`
    Score float64 `json:"score" example:"0.73"`
}

type SimilarityService interface {
	CreateSchemas(ctx context.Context) error
	EncodePost(ctx context.Context, id int) error
	BatchEncodePosts(ctx context.Context, ids []int, batchSize int) ([]int, error)
	FindSimilarPosts(ctx context.Context, filter *PostSimilarityFilter) ([]*Post, error)
	FindSimilarPostIDs(ctx context.Context, filter *PostSimilarityFilter) ([]int, error)
	DeletePost(ctx context.Context, id int) error
}

// used to enable encoding in http request
// only used to bypass encoding when running tests
type ContextKey string

const EncodeContextKey ContextKey = "encode"

// OrderPostsByIDs returns the posts in the order of ids, skipping ids with no post
// and emitting each post at most once
func OrderPostsByIDs(posts []*Post, ids []int) []*Post {
	byID := make(map[int]*Post, len(posts))
	for _, p := range posts {
		byID[p.Id] = p
	}
	ordered := make([]*Post, 0, len(posts))
	for _, id := range ids {
		if p, ok := byID[id]; ok {
			ordered = append(ordered, p)
			delete(byID, id)
		}
	}
	return ordered
}

// DedupeIDs returns ids with repeats removed, keeping first-seen order
func DedupeIDs(ids []int) []int {
	seen := make(map[int]struct{}, len(ids))
	deduped := make([]int, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		deduped = append(deduped, id)
	}
	return deduped
}

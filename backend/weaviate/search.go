package weaviate

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/evanofslack/analogdb"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const searchAlpha = 0.6

var searchProperties = []string{"tags^3", "caption^2", "title"}

var _ analogdb.SearchService = (*SearchService)(nil)

type SearchService struct {
	db *DB
}

func NewSearchService(db *DB) *SearchService {
	return &SearchService{db: db}
}

func (ss SearchService) SearchText(ctx context.Context, filter *analogdb.SearchFilter) ([]analogdb.SearchHit, bool, error) {
	return ss.db.searchText(ctx, filter)
}

func (ss SearchService) SearchImage(ctx context.Context, filter *analogdb.SearchFilter) ([]analogdb.SearchHit, error) {
	return ss.db.searchImage(ctx, filter)
}

func searchFields(additional string) []graphql.Field {
	return []graphql.Field{
		{Name: "post_id"},
		{Name: "tags"},
		{Name: "_additional", Fields: []graphql.Field{{Name: additional}}},
	}
}

// searchText runs a hybrid query, the expanded text goes to BM25 and the
// original text to CLIP. It fetches one extra hit to learn if there is more.
func (db *DB) searchText(ctx context.Context, filter *analogdb.SearchFilter) ([]analogdb.SearchHit, bool, error) {
	ctx, span := db.startTrace(ctx, "vector:search_text", trace.WithAttributes(attribute.String("query", filter.Query)))
	defer span.End()

	if filter.Limit < 1 {
		return nil, false, fmt.Errorf("search limit must be positive")
	}

	nearText := db.db.GraphQL().NearTextArgBuilder().WithConcepts([]string{filter.Query})
	hybrid := db.db.GraphQL().HybridArgumentBuilder().
		WithQuery(filter.Expanded).
		WithAlpha(searchAlpha).
		WithFusionType(graphql.RelativeScore).
		WithProperties(searchProperties).
		WithSearches(db.db.GraphQL().HybridSearchesArgumentBuilder().WithNearText(nearText))

	query := db.db.GraphQL().Get().
		WithClassName(PostImageClass).
		WithFields(searchFields("score")...).
		WithHybrid(hybrid).
		WithLimit(filter.Limit + 1).
		WithOffset(filter.Offset)
	if where := flagsWhere(filter.Nsfw, filter.Grayscale, filter.Sprocket, nil); where != nil {
		query = query.WithWhere(where)
	}

	result, err := query.Do(ctx)
	if err == nil {
		var hits []analogdb.SearchHit
		hits, err = unmarshallSearchResp(result)
		if err == nil {
			hasMore := len(hits) > filter.Limit
			hits = trimHits(hits, filter.Limit)
			return hits, hasMore, nil
		}
	}
	db.logger.ErrorContext(ctx, "Fail search text in vector db", "query", filter.Query, "error", err)
	span.SetStatus(codes.Error, "Search text failed")
	span.RecordError(err)
	return nil, false, err
}

func (db *DB) searchImage(ctx context.Context, filter *analogdb.SearchFilter) ([]analogdb.SearchHit, error) {
	ctx, span := db.startTrace(ctx, "vector:search_image")
	defer span.End()

	if filter.Limit < 1 {
		return nil, fmt.Errorf("search limit must be positive")
	}
	if len(filter.Image) == 0 {
		return nil, fmt.Errorf("search image must not be empty")
	}

	nearImage := db.db.GraphQL().NearImageArgBuilder().
		WithImage(base64.StdEncoding.EncodeToString(filter.Image))
	query := db.db.GraphQL().Get().
		WithClassName(PostImageClass).
		WithFields(searchFields("distance")...).
		WithNearImage(nearImage).
		WithLimit(filter.Limit)
	if where := flagsWhere(filter.Nsfw, filter.Grayscale, filter.Sprocket, nil); where != nil {
		query = query.WithWhere(where)
	}

	result, err := query.Do(ctx)
	if err == nil {
		var hits []analogdb.SearchHit
		hits, err = unmarshallSearchResp(result)
		if err == nil {
			return trimHits(hits, filter.Limit), nil
		}
	}
	db.logger.ErrorContext(ctx, "Fail search image in vector db", "error", err)
	span.SetStatus(codes.Error, "Search image failed")
	span.RecordError(err)
	return nil, err
}

// trimHits drops repeated posts and trims to limit
func trimHits(hits []analogdb.SearchHit, limit int) []analogdb.SearchHit {
	seen := make(map[int]struct{}, len(hits))
	out := make([]analogdb.SearchHit, 0, len(hits))
	for _, hit := range hits {
		if _, ok := seen[hit.PostID]; ok {
			continue
		}
		seen[hit.PostID] = struct{}{}
		out = append(out, hit)
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func unmarshallSearchResp(result *models.GraphQLResponse) ([]analogdb.SearchHit, error) {
	hits := []analogdb.SearchHit{}

	if result == nil {
		return hits, errors.New("unmarshall search from vector DB: empty response")
	}
	if len(result.Errors) > 0 {
		return hits, fmt.Errorf("unmarshall search from vector DB: %s", result.Errors[0].Message)
	}

	data, ok := result.Data["Get"].(map[string]interface{})
	if !ok {
		return hits, errors.New("unmarshall search from vector DB: missing Get")
	}

	pictures, _ := data[PostImageClass].([]interface{})
	for _, picture := range pictures {
		fields, ok := picture.(map[string]interface{})
		if !ok {
			continue
		}
		postID, ok := fields["post_id"].(float64)
		if !ok {
			continue
		}
		hit := analogdb.SearchHit{PostID: int(postID), Tags: []string{}}
		if tags, ok := fields["tags"].([]interface{}); ok {
			for _, tag := range tags {
				if t, ok := tag.(string); ok {
					hit.Tags = append(hit.Tags, t)
				}
			}
		}
		hits = append(hits, hit)
	}
	return hits, nil
}

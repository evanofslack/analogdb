package weaviate

import (
	"context"
	"errors"
	"fmt"

	"github.com/evanofslack/analogdb"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/data/replication"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const PostImageClass = "PostImage"

// some posts have more than one picture object, so fetch extra
// neighbors to still fill the limit after removing repeats
const (
	similarOverFetch = 3
	maxSimilarFetch  = 200
)

var _ analogdb.SimilarityService = (*SimilarityService)(nil)

type SimilarityService struct {
	db          *DB
	postService analogdb.PostService
}

func NewSimilarityService(db *DB, ps analogdb.PostService) *SimilarityService {
	return &SimilarityService{db: db, postService: ps}
}

func (ss SimilarityService) DeletePost(ctx context.Context, postID int) error {
	return ss.db.deletePost(ctx, postID)
}

func (ss SimilarityService) FindSimilarPosts(ctx context.Context, similarityFilter *analogdb.PostSimilarityFilter) ([]*analogdb.Post, error) {
	ctx, span := ss.db.tracer.Tracer.Start(ctx, "vector:find_similar_posts")
	defer span.End()

	// get similar IDs
	ids, err := ss.db.getSimilarPostIDs(ctx, similarityFilter)
	if err != nil {
		return nil, err
	}

	// turn IDs into posts
	filter := analogdb.NewPostFilterWithIDs(ids)
	posts, _, err := ss.postService.FindPosts(ctx, filter)
	if err != nil {
		return nil, err
	}
	return analogdb.OrderPostsByIDs(posts, ids), nil
}

func (ss SimilarityService) FindSimilarPostIDs(ctx context.Context, similarityFilter *analogdb.PostSimilarityFilter) ([]int, error) {
	ctx, span := ss.db.tracer.Tracer.Start(ctx, "vector:find_similar_post_ids")
	defer span.End()

	return ss.db.getSimilarPostIDs(ctx, similarityFilter)
}

func (db *DB) deletePost(ctx context.Context, id int) error {
	db.logger.DebugContext(ctx, "Start delete post from vector DB", "post_id", id)
	ctx, span := db.startTrace(ctx, "vector:delete_post", trace.WithAttributes(attribute.Int("postID", id)))
	defer span.End()

	where := filters.Where().
		WithPath([]string{"post_id"}).
		WithOperator(filters.Equal).
		WithValueInt(int64(id))

	result, err := db.db.Batch().ObjectsBatchDeleter().
		WithClassName(PostImageClass).
		WithWhere(where).
		WithOutput("minimal").
		WithConsistencyLevel(replication.ConsistencyLevel.ALL). // default QUORUM
		Do(ctx)
	if err != nil || result == nil || result.Results == nil {
		err = fmt.Errorf("delete postID from vector DB, err=%w", err)
		db.logger.ErrorContext(ctx, "Fail delete post from vector db", "post_id", id, "error", err)
		span.SetStatus(codes.Error, "Delete picture failed")
		span.RecordError(err)
		return &analogdb.Error{Code: analogdb.ERRINTERNAL, Message: fmt.Sprintf("post %d could not be deleted from vector DB", id)}
	}

	results := result.Results
	if results.Matches == 0 {
		db.logger.WarnContext(ctx, "Found no post to delete in vector db", "post_id", id)
		span.SetStatus(codes.Error, "Post not found")
		return &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: fmt.Sprintf("post %d not found", id)}
	}
	if results.Failed > 0 {
		err = fmt.Errorf("failed to delete %d of %d objects", results.Failed, results.Matches)
		db.logger.ErrorContext(ctx, "Fail delete post from vector db", "post_id", id, "error", err)
		span.SetStatus(codes.Error, "Delete picture failed")
		span.RecordError(err)
		return &analogdb.Error{Code: analogdb.ERRINTERNAL, Message: fmt.Sprintf("post %d could not be deleted from vector DB", id)}
	}

	span.AddEvent("Deleted pictures", trace.WithAttributes(attribute.Int("postID", id), attribute.Int64("count", results.Successful)))
	db.logger.InfoContext(ctx, "Finish delete post from vector db", "post_id", id, "count", results.Successful)
	return nil
}

type pictureResponse struct {
	postID   int
	distance float64
	uuid     string
}

func (db *DB) getSimilarPostIDs(ctx context.Context, filter *analogdb.PostSimilarityFilter) ([]int, error) {
	var ids []int

	if filter.ID == nil {
		return ids, fmt.Errorf("postID cannot be nil")
	}

	postID := *filter.ID
	db.logger.DebugContext(ctx, "Start get similar posts from vector db", "post_id", postID)
	ctx, span := db.startTrace(ctx, "vector:get_similar_post_ids", trace.WithAttributes(attribute.Int("postID", postID)))
	defer span.End()

	fields := []graphql.Field{
		{Name: "post_id"},
		{Name: "_additional", Fields: []graphql.Field{
			{Name: "distance"},
			{Name: "id"},
		}},
	}

	uuid := pictureID(postID).String()
	exists, err := db.db.Data().Checker().
		WithClassName(PostImageClass).
		WithID(uuid).
		Do(ctx)
	if err != nil {
		db.logger.ErrorContext(ctx, "Fail check embedding exists in vector db", "post_id", postID, "error", err)
		span.SetStatus(codes.Error, "Check embedding exists failed")
		span.RecordError(err)
		return ids, err
	}
	if !exists {
		db.logger.WarnContext(ctx, "Found no embedding in vector db", "post_id", postID)
		span.SetStatus(codes.Error, "Embedding not found")
		return ids, &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: fmt.Sprintf("post %d has no similarity embedding", postID)}
	}

	// find nearest neighbors

	// this is where we narrow down the results
	where, err := filterToWhere(filter)
	if err != nil {
		db.logger.ErrorContext(ctx, "Fail convert similarity filter to where clause", "post_id", postID, "error", err)
		span.SetStatus(codes.Error, "Similarity filter to where clause failed")
		span.RecordError(err)
		return ids, err
	}

	// and set the limit
	var limit int
	if lim := filter.Limit; lim != nil {
		limit = *lim
	}

	nearObject := db.db.GraphQL().NearObjectArgBuilder().WithID(uuid)
	query := db.db.GraphQL().Get().
		WithClassName(PostImageClass).
		WithFields(fields...).
		WithNearObject(nearObject)
	if limit > 0 {
		query = query.WithLimit(fetchLimit(limit))
	}
	if where != nil {
		query = query.WithWhere(where)
	}
	result, err := query.Do(ctx)
	if err != nil {
		db.logger.ErrorContext(ctx, "Fail find near embeddings in vector db", "post_id", postID, "error", err)
		span.SetStatus(codes.Error, "Failed to find similar embeddings in vector DB")
		span.RecordError(err)
		return ids, err
	}
	span.AddEvent("Found similar embeddings", trace.WithAttributes(attribute.Int("postID", postID), attribute.String("uuid", uuid)))

	pics, err := unmarshallPicturesResp(result)
	if err != nil {
		db.logger.ErrorContext(ctx, "Fail unmarshall post from vector db", "post_id", postID, "error", err)
		span.SetStatus(codes.Error, "Unmarshall embedding failed")
		span.RecordError(err)
		return ids, err
	}
	span.AddEvent("Unmarshalled embedding", trace.WithAttributes(attribute.Int("postID", postID), attribute.String("uuid", uuid)))

	ids = similarPostIDs(pics, postID, limit)

	if len(ids) == 0 {
		db.logger.WarnContext(ctx, "Found zero similar posts", "post_id", postID)
		span.SetStatus(codes.Error, "Found zero similar posts")
		span.RecordError(err)
		return ids, &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "no similar posts found"}
	}
	return ids, err
}

func fetchLimit(limit int) int {
	if limit <= 0 {
		return limit
	}
	return min(limit*similarOverFetch, max(limit, maxSimilarFetch))
}

// similarPostIDs returns the post ids of pics in order, without repeats
// or the post itself, trimmed to limit
func similarPostIDs(pics []pictureResponse, postID int, limit int) []int {
	ids := make([]int, 0, len(pics))
	for _, pic := range pics {
		if pic.postID != postID {
			ids = append(ids, pic.postID)
		}
	}
	ids = analogdb.DedupeIDs(ids)
	if limit > 0 && len(ids) > limit {
		ids = ids[:limit]
	}
	return ids
}

func filterToWhere(filter *analogdb.PostSimilarityFilter) (*filters.WhereBuilder, error) {
	statements := []*filters.WhereBuilder{}

	if nsfw := filter.Nsfw; nsfw != nil {
		statements = append(statements,
			filters.Where().
				WithPath([]string{"nsfw"}).
				WithOperator(filters.Equal).
				WithValueBoolean(*nsfw),
		)
	}
	if sprocket := filter.Sprocket; sprocket != nil {
		statements = append(statements,
			filters.Where().
				WithPath([]string{"sprocket"}).
				WithOperator(filters.Equal).
				WithValueBoolean(*sprocket),
		)
	}
	if grayscale := filter.Grayscale; grayscale != nil {
		statements = append(statements,
			filters.Where().
				WithPath([]string{"grayscale"}).
				WithOperator(filters.Equal).
				WithValueBoolean(*grayscale),
		)
	}
	if exclude := filter.ExcludeIDs; exclude != nil {
		for _, excludeID := range *exclude {
			statements = append(statements,
				filters.Where().
					WithPath([]string{"post_id"}).
					WithOperator(filters.NotEqual).
					WithValueInt(int64(excludeID)),
			)
		}
	}

	if len(statements) == 0 {
		return nil, nil
	}
	where := filters.Where().
		WithOperator(filters.And).
		WithOperands(statements)
	return where, nil
}

func unmarshallPicturesResp(result *models.GraphQLResponse) ([]pictureResponse, error) {
	var picturesResponse []pictureResponse

	if result == nil {
		return picturesResponse, errors.New("unmarshall pictures from vector DB: empty response")
	}
	if len(result.Errors) > 0 {
		return picturesResponse, fmt.Errorf("unmarshall pictures from vector DB: %s", result.Errors[0].Message)
	}

	data, ok := result.Data["Get"].(map[string]interface{})
	if !ok {
		return picturesResponse, errors.New("unmarshall pictures from vector DB: missing Get")
	}

	pictures, _ := data[PostImageClass].([]interface{})
	for _, picture := range pictures {
		var pic pictureResponse
		if fields, ok := picture.(map[string]interface{}); ok {
			if postID, ok := fields["post_id"].(float64); ok {
				pic.postID = int(postID)
			}
			if additional, ok := fields["_additional"].(map[string]interface{}); ok {
				if distance, ok := additional["distance"].(float64); ok {
					pic.distance = distance
				}
				if uuid, ok := additional["id"].(string); ok {
					pic.uuid = uuid
				}
			}
		}
		picturesResponse = append(picturesResponse, pic)
	}
	return picturesResponse, nil
}

package weaviate

import (
	"context"
	"errors"
	"fmt"

	"github.com/evanofslack/analogdb"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/data/replication"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const PictureClass = "Picture"

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

	var posts []*analogdb.Post

	// get similar IDs
	ids, err := ss.db.getSimilarPostIDs(ctx, similarityFilter)
	if err != nil {
		return nil, err
	}

	// turn IDs into posts
	filter := analogdb.NewPostFilter(nil, nil, nil, nil, nil, nil, nil, &ids, nil, nil, nil, nil)
	posts, _, err = ss.postService.FindPosts(ctx, filter)
	return posts, err
}

func (db *DB) deletePost(ctx context.Context, postID int) error {

	db.logger.Debug().Ctx(ctx).Int("postID", postID).Msg("Starting delete post from vector DB")

	ctx, span := db.startTrace(ctx, "vector:delete_post", trace.WithAttributes(attribute.Int("postID", postID)))
	defer span.End()

	fields := []graphql.Field{
		{Name: "post_id"},
		{Name: "_additional", Fields: []graphql.Field{
			{Name: "distance"},
			{Name: "id"},
		}},
	}

	where := filters.Where().
		WithPath([]string{"post_id"}).
		WithOperator(filters.Equal).
		WithValueInt(int64(postID))

	result, err := db.db.GraphQL().Get().
		WithClassName(PictureClass).
		WithFields(fields...).
		WithLimit(1).
		WithWhere(where).
		Do(ctx)

	if err != nil || result == nil {
		err = fmt.Errorf("Failed to find postID in vector DB, err=%w", err)
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", postID).Msg("Failed to delete post from vectorDB")
		span.SetStatus(codes.Error, "Get embedding by postID failed")
		span.RecordError(err)
		return &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: fmt.Sprintf("Post %d not found", postID)}
	}
	span.AddEvent("Got vector embedding by postID", trace.WithAttributes(attribute.Int("postID", postID)))

	pics, err := unmarshallPicturesResp(result)
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", postID).Msg("Failed to delete post from vector DB")
		span.SetStatus(codes.Error, "Unmarshall embedding failed")
		span.RecordError(err)
		return &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: fmt.Sprintf("Post %d not found", postID)}
	}
	uuid := pics[0].uuid
	span.AddEvent("Unmarshalled embedding", trace.WithAttributes(attribute.Int("postID", postID), attribute.String("uuid", uuid)))

	err = db.db.Data().Deleter().
		WithClassName(PictureClass).
		WithID(pics[0].uuid).
		WithConsistencyLevel(replication.ConsistencyLevel.ALL). // default QUORUM
		Do(ctx)

	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", postID).Msg("Failed to delete post from vector DB")
		span.SetStatus(codes.Error, "Delete picture failed")
		span.RecordError(err)
		return &analogdb.Error{Code: analogdb.ERRINTERNAL, Message: fmt.Sprintf("Post %d could not be deleted from vector DB", postID)}
	}
	span.AddEvent("Deleted picture", trace.WithAttributes(attribute.Int("postID", postID), attribute.String("uuid", uuid)))

	db.logger.Info().Ctx(ctx).Int("postID", postID).Msg("Deleted post from vector DB")

	return err
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

	db.logger.Debug().Ctx(ctx).Int("postID", postID).Msg("Starting get similar posts from vector DB")

	ctx, span := db.startTrace(ctx, "vector:get_similar_post_ids", trace.WithAttributes(attribute.Int("postID", postID)))
	defer span.End()

	// first make the query to lookup UUID associated with post's embedding

	fields := []graphql.Field{
		{Name: "post_id"},
		{Name: "_additional", Fields: []graphql.Field{
			{Name: "distance"},
			{Name: "id"},
		}},
	}

	where := filters.Where().
		WithPath([]string{"post_id"}).
		WithOperator(filters.Equal).
		WithValueInt(int64(postID))

	result, err := db.db.GraphQL().Get().
		WithClassName("Picture").
		WithFields(fields...).
		WithLimit(1).
		WithWhere(where).
		Do(ctx)

	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", postID).Msg("Failed to find post in vector DB")
		span.SetStatus(codes.Error, "Get embedding by postID failed")
		span.RecordError(err)
		return ids, err
	}
	span.AddEvent("Got vector embedding by postID", trace.WithAttributes(attribute.Int("postID", postID)))

	pics, err := unmarshallPicturesResp(result)
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", postID).Msg("Failed to unmarshall post from vector DB")
		span.SetStatus(codes.Error, "Unmarshall embedding failed")
		span.RecordError(err)
		return ids, &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: fmt.Sprintf("Post %d not found", postID)}
	}
	uuid := pics[0].uuid
	span.AddEvent("Unmarshalled embedding", trace.WithAttributes(attribute.Int("postID", postID), attribute.String("uuid", uuid)))

	// then make query to find nearest neighbors

	// this is where we narrow down the results
	where, err = filterToWhere(filter)
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", postID).Msg("Failed to convert similarity filter to where clause")
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
	result, err = db.db.GraphQL().Get().
		WithClassName(PictureClass).
		WithFields(fields...).
		WithLimit(limit).
		WithWhere(where).
		WithNearObject(nearObject).
		Do(ctx)

	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", postID).Msg("Failed to find near embeddings in vector DB")
		span.SetStatus(codes.Error, "Failed to find similar embeddings in vector DB")
		span.RecordError(err)
		return ids, err
	}
	span.AddEvent("Found similar embeddings", trace.WithAttributes(attribute.Int("postID", postID), attribute.String("uuid", uuid)))

	pics, err = unmarshallPicturesResp(result)
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", postID).Msg("Failed to unmarshall post from vector DB")
		span.SetStatus(codes.Error, "Unmarshall embedding failed")
		span.RecordError(err)
		return ids, err
	}
	span.AddEvent("Unmarshalled embedding", trace.WithAttributes(attribute.Int("postID", postID), attribute.String("uuid", uuid)))

	for _, pic := range pics {
		ids = append(ids, pic.postID)
	}

	if len(ids) == 0 {
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", postID).Msg("Found zero similar posts")
		span.SetStatus(codes.Error, "Found zero similar posts")
		span.RecordError(err)
		return ids, &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "No similar posts found"}
	}

	return ids, err
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
				WithPath([]string{"greyscale"}).
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

	where := filters.Where().
		WithOperator(filters.And).
		WithOperands(statements)
	return where, nil

}

func unmarshallPicturesResp(result *models.GraphQLResponse) ([]pictureResponse, error) {

	var picturesResponse []pictureResponse

	data := result.Data["Get"].(map[string]interface{})

	// dear god i hate this
	if pictures, ok := data["Picture"].([]interface{}); ok {
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
	}
	if len(picturesResponse) > 0 {
		return picturesResponse, nil
	}

	err := errors.New("Failed to unmarshall pictures from vector DB")
	return picturesResponse, err
}

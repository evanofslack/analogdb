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

func (ss SimilarityService) DeletePost(ctx context.Context, postID int) (error) {
	return ss.db.deletePost(ctx, postID)
}

func (ss SimilarityService) FindSimilarPostsByImage(ctx context.Context, postID int, similarityFilter *analogdb.PostSimilarityFilter) ([]*analogdb.Post, error) {

	var posts []*analogdb.Post

	// make sure we exclude the post we are query from the results
	excluded := []int{postID}
	similarityFilter.ExcludeIDs = &excluded

	// get similar IDs
	ids, err := ss.db.getSimilarPostIDs(ctx, postID, similarityFilter)
	if err != nil {
		return nil, err
	}

	// turn IDs into posts
	filter := analogdb.PostFilter{IDs: &ids}
	posts, _, err = ss.postService.FindPosts(ctx, &filter)
	return posts, err
}


func (db *DB) deletePost(ctx context.Context, postID int) (error) {

	db.logger.Debug().Int("postID", postID).Msg("Starting delete post from vector DB")
	 
	where := filters.Where().
		WithPath([]string{"post_id"}).
		WithOperator(filters.Equal).
		WithValueInt(int64(postID))

	result, err := db.db.GraphQL().Get().
		WithClassName(PictureClass).
		WithLimit(1).
		WithWhere(where).
		Do(ctx)

	if err != nil {
		err = fmt.Errorf("Failed to find postID in vector DB, err=%w", err)
		db.logger.Error().Err(err).Int("postID", postID).Msg("Failed to delete post from vectorDB")
		return &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: fmt.Sprintf("Post %d not found", postID)}
	}

	pics, err := unmarshallPicturesResp(result)
	if err != nil {
		db.logger.Error().Err(err).Int("postID", postID).Msg("Failed to delete post from vector DB")
		return &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: fmt.Sprintf("Post %d not found", postID)}
	}

	err = db.db.Data().Deleter().
		WithClassName(PictureClass).
		WithID(pics[0].uuid).
		WithConsistencyLevel(replication.ConsistencyLevel.ALL).  // default QUORUM
		Do(ctx)

  if err != nil {
	db.logger.Error().Err(err).Int("postID", postID).Msg("Failed to delete post from vector DB")
	return &analogdb.Error{Code: analogdb.ERRINTERNAL, Message: fmt.Sprintf("Post %d could not be deleted from vector DB", postID)}
  }

	db.logger.Info().Int("postID", postID).Msg("Deleted post from vector DB")

	return err
}

type pictureResponse struct {
	postID   int
	distance float64
	uuid     string
}

func (db *DB) getSimilarPostIDs(ctx context.Context, postID int, filter *analogdb.PostSimilarityFilter) ([]int, error) {

	db.logger.Debug().Int("postID", postID).Msg("Starting get similar posts from vector DB")

	var ids []int

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
		db.logger.Error().Err(err).Int("postID", postID).Msg("Failed to find post in vector DB")
		return ids, err
	}

	pics, err := unmarshallPicturesResp(result)
	if err != nil {
		db.logger.Error().Err(err).Int("postID", postID).Msg("Failed to unmarshall post from vector DB")
		return ids, &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: fmt.Sprintf("Post %d not found", postID)}
	}

	// then make query to find nearest neighbors

	// this is where we narrow down the results
	where, err = filterToWhere(filter)
	if err != nil {
		db.logger.Error().Err(err).Int("postID", postID).Msg("Failed to convert similarity filter to where clause")
		return ids, err
	}

	// and set the limit
	var limit int
	if lim := filter.Limit; lim != nil {
		limit = *lim
	}

	nearObject := db.db.GraphQL().NearObjectArgBuilder().WithID(pics[0].uuid)
	result, err = db.db.GraphQL().Get().
		WithClassName("Picture").
		WithFields(fields...).
		WithLimit(limit).
		WithWhere(where).
		WithNearObject(nearObject).
		Do(ctx)

	if err != nil {
		db.logger.Error().Err(err).Int("postID", postID).Msg("Failed to find near posts in vector DB")
		return ids, err
	}

	pics, err = unmarshallPicturesResp(result)
	if err != nil {
		db.logger.Error().Err(err).Int("postID", postID).Msg("Failed to unmarshall post from vector DB")
		return ids, err
	}

	for _, pic := range pics {
		ids = append(ids, pic.postID)
	}

	if len(ids) == 0 {
		db.logger.Error().Err(err).Int("postID", postID).Msg("Found zero similar posts")
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

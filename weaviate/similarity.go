package weaviate

import (
	"context"
	"errors"
	"fmt"

	"github.com/evanofslack/analogdb"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

var _ analogdb.SimilarityService = (*SimilarityService)(nil)

type SimilarityService struct {
	db          *DB
	postService analogdb.PostService
}

func NewSimilarityService(db *DB, ps analogdb.PostService) *SimilarityService {
	return &SimilarityService{db: db, postService: ps}
}

func (ss SimilarityService) FindSimilarPostsByImage(ctx context.Context, postID int, limit int) ([]*analogdb.Post, error) {

	var posts []*analogdb.Post

	ids, err := ss.db.getSimilarPostIDs(ctx, postID, limit)
	if err != nil {
		return nil, err
	}

	filter := analogdb.PostFilter{IDs: &ids}

	posts, _, err = ss.postService.FindPosts(ctx, &filter)
	return posts, err
}

type pictureResponse struct {
	postID   int
	distance float64
	uuid     string
}

func (db *DB) getSimilarPostIDs(ctx context.Context, postID int, limit int) ([]int, error) {

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
		WithFields(fields...).
		WithWhere(where).
		Do(ctx)

	if err != nil {
		return ids, err
	}

	pics, err := unmarshallPicturesResp(result)

	// then make query to find nearest neighbors
	nearObject := db.db.GraphQL().NearObjectArgBuilder().WithID(pics[0].uuid)
	result, err = db.db.GraphQL().Get().
		WithClassName("Picture").
		WithFields(fields...).
		WithLimit(limit).
		WithFields(fields...).
		WithNearObject(nearObject).
		Do(ctx)

	if err != nil {
		return ids, err
	}

	pics, err = unmarshallPicturesResp(result)
	if err != nil {
		return ids, err
	}

	for _, pic := range pics {
		ids = append(ids, pic.postID)
	}

	return ids, err
}

func unmarshallPicturesResp(result *models.GraphQLResponse) ([]pictureResponse, error) {

	var picturesResponse []pictureResponse

	fmt.Println(result)
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

	return picturesResponse, errors.New("Failed to unmarshall pictures from vector database")

}

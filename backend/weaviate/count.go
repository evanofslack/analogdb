package weaviate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/evanofslack/analogdb"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

// ensure interface is implemented
var _ analogdb.VectorCounter = (*DB)(nil)

func (db *DB) CountObjects(ctx context.Context) (int, error) {
	meta := graphql.Field{Name: "meta", Fields: []graphql.Field{{Name: "count"}}}
	result, err := db.db.GraphQL().Aggregate().
		WithClassName(PictureClass).
		WithFields(meta).
		Do(ctx)
	if err != nil {
		return 0, err
	}
	return unmarshallCountResp(result)
}

func unmarshallCountResp(result *models.GraphQLResponse) (int, error) {
	if len(result.Errors) > 0 {
		return 0, fmt.Errorf("aggregate count: %s", result.Errors[0].Message)
	}
	data, err := json.Marshal(result.Data)
	if err != nil {
		return 0, err
	}
	var resp struct {
		Aggregate map[string][]struct {
			Meta struct {
				Count int `json:"count"`
			} `json:"meta"`
		} `json:"Aggregate"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return 0, err
	}
	classes := resp.Aggregate[PictureClass]
	if len(classes) == 0 {
		return 0, fmt.Errorf("aggregate count: no result for class %s", PictureClass)
	}
	return classes[0].Meta.Count, nil
}

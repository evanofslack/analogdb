package weaviate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/evanofslack/analogdb"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

// ensure interface is implemented
var (
	_ analogdb.VectorCounter = (*DB)(nil)
	_ analogdb.VectorLister  = (*DB)(nil)
)

const vectorIDsPageSize = 2000

func (db *DB) CountObjects(ctx context.Context) (int, error) {
	meta := graphql.Field{Name: "meta", Fields: []graphql.Field{{Name: "count"}}}
	result, err := db.db.GraphQL().Aggregate().
		WithClassName(PostImageClass).
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
	classes := resp.Aggregate[PostImageClass]
	if len(classes) == 0 {
		return 0, fmt.Errorf("aggregate count: no result for class %s", PostImageClass)
	}
	return classes[0].Meta.Count, nil
}

// VectorPostIDs pages through every object with the cursor API and returns
// the distinct post ids
func (db *DB) VectorPostIDs(ctx context.Context) ([]int, error) {
	fields := []graphql.Field{
		{Name: "post_id"},
		{Name: "_additional", Fields: []graphql.Field{{Name: "id"}}},
	}
	seen := map[int]struct{}{}
	ids := []int{}
	after := ""
	for {
		query := db.db.GraphQL().Get().
			WithClassName(PostImageClass).
			WithFields(fields...).
			WithLimit(vectorIDsPageSize)
		if after != "" {
			query = query.WithAfter(after)
		}
		result, err := query.Do(ctx)
		if err != nil {
			return nil, err
		}
		postIDs, last, err := unmarshallIDsResp(result)
		if err != nil {
			return nil, err
		}
		if last == "" {
			break
		}
		for _, id := range postIDs {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
		after = last
	}
	slices.Sort(ids)
	return ids, nil
}

// unmarshallIDsResp returns the post ids of one page and the object id of its last object
func unmarshallIDsResp(result *models.GraphQLResponse) ([]int, string, error) {
	if result == nil {
		return nil, "", errors.New("list vector ids: empty response")
	}
	if len(result.Errors) > 0 {
		return nil, "", fmt.Errorf("list vector ids: %s", result.Errors[0].Message)
	}
	data, err := json.Marshal(result.Data)
	if err != nil {
		return nil, "", err
	}
	var resp struct {
		Get map[string][]struct {
			PostID     *int `json:"post_id"`
			Additional struct {
				ID string `json:"id"`
			} `json:"_additional"`
		} `json:"Get"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, "", err
	}
	objects := resp.Get[PostImageClass]
	ids := make([]int, 0, len(objects))
	last := ""
	for _, obj := range objects {
		last = obj.Additional.ID
		if obj.PostID != nil {
			ids = append(ids, *obj.PostID)
		}
	}
	return ids, last, nil
}

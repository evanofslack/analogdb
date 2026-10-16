package weaviate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/evanofslack/analogdb"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

// ensure interface is implemented
var _ analogdb.VectorScanner = (*DB)(nil)

const vectorScanPageSize = 500

// ScanVectors pages through every object with the cursor API and calls fn with
// its post id and vector. The image blob is never read.
func (db *DB) ScanVectors(ctx context.Context, fn func(postID int, vector []float32) error) error {
	fields := []graphql.Field{
		{Name: "post_id"},
		{Name: "_additional", Fields: []graphql.Field{{Name: "id"}, {Name: "vector"}}},
	}
	after := ""
	for {
		query := db.db.GraphQL().Get().
			WithClassName(PostImageClass).
			WithFields(fields...).
			WithLimit(vectorScanPageSize)
		if after != "" {
			query = query.WithAfter(after)
		}
		result, err := query.Do(ctx)
		if err != nil {
			return err
		}
		objects, err := unmarshallVectorsResp(result)
		if err != nil {
			return err
		}
		if len(objects) == 0 {
			return nil
		}
		for _, obj := range objects {
			if obj.PostID == nil || len(obj.Additional.Vector) == 0 {
				continue
			}
			if err := fn(*obj.PostID, obj.Additional.Vector); err != nil {
				return err
			}
		}
		after = objects[len(objects)-1].Additional.ID
	}
}

type vectorObject struct {
	PostID     *int `json:"post_id"`
	Additional struct {
		ID     string    `json:"id"`
		Vector []float32 `json:"vector"`
	} `json:"_additional"`
}

func unmarshallVectorsResp(result *models.GraphQLResponse) ([]vectorObject, error) {
	if result == nil {
		return nil, errors.New("scan vectors: empty response")
	}
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("scan vectors: %s", result.Errors[0].Message)
	}
	data, err := json.Marshal(result.Data)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Get map[string][]vectorObject `json:"Get"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Get[PostImageClass], nil
}

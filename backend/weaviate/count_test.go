package weaviate

import (
	"encoding/json"
	"testing"

	"github.com/weaviate/weaviate/entities/models"
)

func TestUnmarshallCountResp(t *testing.T) {
	var result models.GraphQLResponse
	body := `{"data":{"Aggregate":{"PostImage":[{"meta":{"count":23410}}]}}}`
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatal(err)
	}
	count, err := unmarshallCountResp(&result)
	if err != nil {
		t.Fatal(err)
	}
	if count != 23410 {
		t.Errorf("want 23410, got %d", count)
	}

	var empty models.GraphQLResponse
	if err := json.Unmarshal([]byte(`{"data":{"Aggregate":{"PostImage":[]}}}`), &empty); err != nil {
		t.Fatal(err)
	}
	if _, err := unmarshallCountResp(&empty); err == nil {
		t.Error("want error for empty result")
	}
}

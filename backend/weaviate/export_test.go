package weaviate

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/weaviate/weaviate/entities/models"
)

func TestUnmarshallVectorsResp(t *testing.T) {
	var result models.GraphQLResponse
	body := `{"data":{"Get":{"PostImage":[
		{"post_id":7,"_additional":{"id":"a","vector":[0.6,0.8]}},
		{"post_id":null,"_additional":{"id":"b","vector":[1,0]}}
	]}}}`
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatal(err)
	}
	objects, err := unmarshallVectorsResp(&result)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 2 || *objects[0].PostID != 7 || objects[0].Additional.Vector[1] != 0.8 {
		t.Errorf("want post 7 with its vector, got %+v", objects)
	}
	if objects[1].PostID != nil || objects[1].Additional.ID != "b" {
		t.Errorf("want an object without a post id kept for paging, got %+v", objects[1])
	}

	var failed models.GraphQLResponse
	if err := json.Unmarshal([]byte(`{"errors":[{"message":"class not found"}]}`), &failed); err != nil {
		t.Fatal(err)
	}
	if _, err := unmarshallVectorsResp(&failed); err == nil || !strings.Contains(err.Error(), "class not found") {
		t.Errorf("want graphql error, got %v", err)
	}
}

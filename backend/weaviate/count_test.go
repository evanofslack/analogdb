package weaviate

import (
	"encoding/json"
	"slices"
	"strings"
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

func TestUnmarshallIDsResp(t *testing.T) {
	var result models.GraphQLResponse
	body := `{"data":{"Get":{"PostImage":[
		{"post_id":7,"_additional":{"id":"a"}},
		{"post_id":9,"_additional":{"id":"b"}},
		{"post_id":7,"_additional":{"id":"c"}}
	]}}}`
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatal(err)
	}
	ids, last, err := unmarshallIDsResp(&result)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(ids, []int{7, 9, 7}) || last != "c" {
		t.Errorf("want ids [7 9 7] and last c, got %v and %q", ids, last)
	}

	var empty models.GraphQLResponse
	if err := json.Unmarshal([]byte(`{"data":{"Get":{"PostImage":[]}}}`), &empty); err != nil {
		t.Fatal(err)
	}
	ids, last, err = unmarshallIDsResp(&empty)
	if err != nil || len(ids) != 0 || last != "" {
		t.Errorf("want empty page, got %v, %q, %v", ids, last, err)
	}

	var failed models.GraphQLResponse
	if err := json.Unmarshal([]byte(`{"errors":[{"message":"class not found"}]}`), &failed); err != nil {
		t.Fatal(err)
	}
	if _, _, err := unmarshallIDsResp(&failed); err == nil || !strings.Contains(err.Error(), "class not found") {
		t.Errorf("want graphql error, got %v", err)
	}
}

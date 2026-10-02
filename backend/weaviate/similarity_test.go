package weaviate

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/evanofslack/analogdb"
	"github.com/weaviate/weaviate/entities/models"
)

func TestSimilarPostIDs(t *testing.T) {
	pics := []pictureResponse{
		{postID: 10485}, {postID: 10485}, {postID: 7}, {postID: 12926},
		{postID: 10485}, {postID: 8}, {postID: 7}, {postID: 9}, {postID: 10},
	}

	tests := []struct {
		name     string
		limit    int
		expected []int
	}{
		{"trims to limit", 3, []int{10485, 7, 8}},
		{"limit larger than results", 10, []int{10485, 7, 8, 9, 10}},
		{"no limit", 0, []int{10485, 7, 8, 9, 10}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := similarPostIDs(pics, 12926, tt.limit)
			if !slices.Equal(got, tt.expected) {
				t.Errorf("similar ids = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFetchLimit(t *testing.T) {
	tests := []struct {
		limit    int
		expected int
	}{
		{0, 0},
		{1, 3},
		{12, 36},
		{50, 150},
		{100, 200},
		{300, 300},
	}

	for _, tt := range tests {
		if got := fetchLimit(tt.limit); got != tt.expected {
			t.Errorf("fetchLimit(%d) = %d, want %d", tt.limit, got, tt.expected)
		}
	}
}

func TestUnmarshallPicturesResp(t *testing.T) {
	var result models.GraphQLResponse
	body := `{"data":{"Get":{"PostImage":[
		{"post_id":7,"_additional":{"distance":0.12,"id":"a"}},
		{"post_id":9,"_additional":{"distance":0.2,"id":"b"}}
	]}}}`
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatal(err)
	}
	pics, err := unmarshallPicturesResp(&result)
	if err != nil {
		t.Fatal(err)
	}
	want := []pictureResponse{{postID: 7, distance: 0.12, uuid: "a"}, {postID: 9, distance: 0.2, uuid: "b"}}
	if !slices.Equal(pics, want) {
		t.Errorf("want %v, got %v", want, pics)
	}

	var empty models.GraphQLResponse
	if err := json.Unmarshal([]byte(`{"data":{"Get":{"PostImage":[]}}}`), &empty); err != nil {
		t.Fatal(err)
	}
	if pics, err := unmarshallPicturesResp(&empty); err != nil || len(pics) != 0 {
		t.Errorf("want no pictures and no error, got %v, %v", pics, err)
	}

	var failed models.GraphQLResponse
	if err := json.Unmarshal([]byte(`{"data":{"Get":{"PostImage":null}},"errors":[{"message":"no object with id"}]}`), &failed); err != nil {
		t.Fatal(err)
	}
	if _, err := unmarshallPicturesResp(&failed); err == nil {
		t.Error("want error for graphql errors")
	}

	if _, err := unmarshallPicturesResp(&models.GraphQLResponse{}); err == nil {
		t.Error("want error for missing Get")
	}
}

func TestFilterToWhere(t *testing.T) {
	nsfw := false
	grayscale := true
	exclude := []int{3, 4}
	filter := &analogdb.PostSimilarityFilter{Nsfw: &nsfw, Grayscale: &grayscale, ExcludeIDs: &exclude}

	where, err := filterToWhere(filter)
	if err != nil {
		t.Fatal(err)
	}
	built := where.Build()
	if built.Operator != "And" {
		t.Fatalf("want And, got %s", built.Operator)
	}
	if len(built.Operands) != 4 {
		t.Fatalf("want 4 operands, got %d", len(built.Operands))
	}
	first := built.Operands[0]
	if !slices.Equal(first.Path, []string{"nsfw"}) || first.Operator != "Equal" || first.ValueBoolean == nil || *first.ValueBoolean {
		t.Errorf("unexpected nsfw operand %+v", first)
	}
	second := built.Operands[1]
	if !slices.Equal(second.Path, []string{"grayscale"}) || second.ValueBoolean == nil || !*second.ValueBoolean {
		t.Errorf("unexpected grayscale operand %+v", second)
	}
	for i, id := range exclude {
		op := built.Operands[2+i]
		if !slices.Equal(op.Path, []string{"post_id"}) || op.Operator != "NotEqual" || op.ValueInt == nil || *op.ValueInt != int64(id) {
			t.Errorf("unexpected exclude operand %+v", op)
		}
	}
}

func TestFilterToWhereEmpty(t *testing.T) {
	where, err := filterToWhere(&analogdb.PostSimilarityFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if where != nil {
		t.Errorf("want no where clause, got %s", where.String())
	}
}

func TestPostImageClass(t *testing.T) {
	class := postImageClass()
	if class.Class != PostImageClass || class.Vectorizer != "multi2vec-clip" {
		t.Fatalf("unexpected class %s with vectorizer %s", class.Class, class.Vectorizer)
	}
	clip := class.ModuleConfig.(map[string]any)["multi2vec-clip"].(map[string]any)
	if fields := clip["imageFields"].([]string); !slices.Equal(fields, []string{"image"}) {
		t.Errorf("want only image vectorized, got %v", fields)
	}
	if _, ok := clip["textFields"]; ok {
		t.Error("want no text fields vectorized")
	}

	types := map[string]string{}
	for _, prop := range class.Properties {
		types[prop.Name] = prop.DataType[0]
		if prop.DataType[0] == "text" || prop.DataType[0] == "text[]" {
			if prop.IndexSearchable == nil || !*prop.IndexSearchable || prop.Tokenization != "word" {
				t.Errorf("want %s searchable with word tokenization", prop.Name)
			}
		}
	}
	want := map[string]string{
		"image": "blob", "post_id": "int", "title": "text", "caption": "text", "tags": "text[]",
		"nsfw": "boolean", "grayscale": "boolean", "sprocket": "boolean",
	}
	if len(types) != len(want) {
		t.Errorf("want %d properties, got %d", len(want), len(types))
	}
	for name, dataType := range want {
		if types[name] != dataType {
			t.Errorf("want %s of type %s, got %q", name, dataType, types[name])
		}
	}
}

package weaviate

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/logger"
	"github.com/weaviate/weaviate/entities/models"
)

func TestBatchBy(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}

	tests := []struct {
		name      string
		batchSize int
		expected  [][]int
	}{
		{"zero batch size", 0, [][]int{{1, 2, 3, 4, 5}}},
		{"negative batch size", -1, [][]int{{1, 2, 3, 4, 5}}},
		{"batch size one", 1, [][]int{{1}, {2}, {3}, {4}, {5}}},
		{"batch size n", 2, [][]int{{1, 2}, {3, 4}, {5}}},
		{"batch size equal to length", 5, [][]int{{1, 2, 3, 4, 5}}},
		{"batch size larger than length", 10, [][]int{{1, 2, 3, 4, 5}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := batchBy(items, tt.batchSize)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func newTestDB(t *testing.T) *DB {
	t.Helper()
	l, err := logger.New("error", "debug", "analogdb-test")
	if err != nil {
		t.Fatal(err)
	}
	return &DB{logger: l}
}

func testPost(id int, url string) *analogdb.Post {
	return &analogdb.Post{
		Id: id,
		DisplayPost: analogdb.DisplayPost{
			Images: []analogdb.Image{{Url: url + "/low"}, {Url: url}},
		},
	}
}

func imageServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Duration(rand.Intn(20)) * time.Millisecond)
		switch {
		case strings.HasPrefix(r.URL.Path, "/missing"):
			http.NotFound(w, r)
		case strings.HasPrefix(r.URL.Path, "/html"):
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<html></html>"))
		default:
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte("image" + r.URL.Path))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestDownloadAndEncodePostsKeepsOrder(t *testing.T) {
	srv := imageServer(t)

	var posts []*analogdb.Post
	for i := 0; i < 50; i++ {
		posts = append(posts, testPost(i, fmt.Sprintf("%s/%d", srv.URL, i)))
	}

	results := downloadAndEncodePosts(context.Background(), posts)
	if len(results) != len(posts) {
		t.Fatalf("want %d results, got %d", len(posts), len(results))
	}
	for i, result := range results {
		if result.err != nil {
			t.Fatalf("post %d: %v", i, result.err)
		}
		if result.post.Id != posts[i].Id {
			t.Errorf("result %d has post %d", i, result.post.Id)
		}
		want := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("image/%d", result.post.Id)))
		if result.image != want {
			t.Errorf("post %d has image of another post", result.post.Id)
		}
	}
}

func TestPostsToPictureObjectsReportsFailures(t *testing.T) {
	srv := imageServer(t)
	db := newTestDB(t)

	posts := []*analogdb.Post{
		testPost(1, srv.URL+"/1"),
		testPost(2, srv.URL+"/missing"),
		testPost(3, srv.URL+"/html"),
		{Id: 4},
		testPost(5, srv.URL+"/5"),
	}

	objects, failed := db.postsToPictureObjects(context.Background(), posts)
	if want := []int{2, 3, 4}; !reflect.DeepEqual(failed, want) {
		t.Errorf("want failed %v, got %v", want, failed)
	}
	if len(objects) != 2 {
		t.Fatalf("want 2 objects, got %d", len(objects))
	}
	for _, obj := range objects {
		postID := obj.Properties.(map[string]interface{})["post_id"].(int)
		if obj.ID != pictureID(postID) {
			t.Errorf("post %d has object ID %s, want %s", postID, obj.ID, pictureID(postID))
		}
	}
}

func TestDownloadAndEncodePostsEmpty(t *testing.T) {
	if results := downloadAndEncodePosts(context.Background(), nil); len(results) != 0 {
		t.Errorf("want no results, got %d", len(results))
	}
	failed, err := SimilarityService{}.BatchEncodePosts(context.Background(), nil, 0)
	if err != nil || len(failed) != 0 {
		t.Errorf("want no failures, got %v, %v", failed, err)
	}
}

func TestPictureIDDeterministic(t *testing.T) {
	a := newPictureObject(&analogdb.Post{Id: 42}, "a")
	b := newPictureObject(&analogdb.Post{Id: 42, DisplayPost: analogdb.DisplayPost{Nsfw: true, Grayscale: true, Sprocket: true}}, "b")
	if a.ID == "" || a.ID != b.ID {
		t.Errorf("want same non-empty ID, got %q and %q", a.ID, b.ID)
	}
	if pictureID(42) == pictureID(43) {
		t.Error("want different IDs for different posts")
	}
}

func TestMissingIDs(t *testing.T) {
	posts := []*analogdb.Post{{Id: 1}, {Id: 3}}
	if got, want := missingIDs([]int{1, 2, 3, 4}, posts), []int{2, 4}; !reflect.DeepEqual(got, want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestNewPictureObject(t *testing.T) {
	caption := "a dog on a beach"
	post := &analogdb.Post{
		Id: 7,
		DisplayPost: analogdb.DisplayPost{
			Title:     "Beach day [Pentax K1000 | Portra 400]",
			Caption:   &caption,
			Nsfw:      true,
			Grayscale: false,
			Sprocket:  true,
			Keywords:  []analogdb.Keyword{{Word: "new york", Weight: 0.9}, {Word: "dog", Weight: 0.5}, {Word: "beach", Weight: 0.1}},
		},
	}

	obj := newPictureObject(post, "aW1hZ2U=")
	if obj.Class != PostImageClass {
		t.Errorf("want class %s, got %s", PostImageClass, obj.Class)
	}
	if obj.ID != pictureID(7) {
		t.Errorf("want id %s, got %s", pictureID(7), obj.ID)
	}
	want := map[string]interface{}{
		"image":     "aW1hZ2U=",
		"post_id":   7,
		"title":     "Beach day [Pentax K1000 | Portra 400]",
		"caption":   "a dog on a beach",
		"tags":      []string{"new york", "dog", "beach"},
		"grayscale": false,
		"nsfw":      true,
		"sprocket":  true,
	}
	if got := obj.Properties.(map[string]interface{}); !reflect.DeepEqual(got, want) {
		t.Errorf("want properties %v, got %v", want, got)
	}
}

func TestNewPictureObjectEmptyCaptionAndTags(t *testing.T) {
	obj := newPictureObject(&analogdb.Post{Id: 1}, "aW1hZ2U=")
	props := obj.Properties.(map[string]interface{})
	if caption, ok := props["caption"].(string); !ok || caption != "" {
		t.Errorf("want empty caption string, got %#v", props["caption"])
	}
	if tags, ok := props["tags"].([]string); !ok || tags == nil || len(tags) != 0 {
		t.Errorf("want empty tags, got %#v", props["tags"])
	}
}

func TestBatchObjectErrors(t *testing.T) {
	failed := models.ObjectsGetResponseAO2ResultStatusFAILED
	success := models.ObjectsGetResponseAO2ResultStatusSUCCESS
	resp := []models.ObjectsGetResponse{
		{Object: models.Object{ID: pictureID(1)}, Result: &models.ObjectsGetResponseAO2Result{Status: &success}},
		{Object: models.Object{ID: pictureID(2)}, Result: &models.ObjectsGetResponseAO2Result{Errors: &models.ErrorResponse{
			Error: []*models.ErrorResponseErrorItems0{{Message: "vectorize image"}, {Message: "timeout"}},
		}}},
		{Object: models.Object{ID: pictureID(3)}},
		{Object: models.Object{ID: pictureID(4)}, Result: &models.ObjectsGetResponseAO2Result{Status: &failed}},
		{Object: models.Object{ID: pictureID(5)}, Result: &models.ObjectsGetResponseAO2Result{Errors: &models.ErrorResponse{}}},
	}

	got := batchObjectErrors(resp)
	want := []objectError{
		{id: pictureID(2), message: "vectorize image, timeout"},
		{id: pictureID(4), message: "status failed"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

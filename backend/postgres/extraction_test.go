package postgres

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/evanofslack/analogdb"
)

func testExtraction(postID int, unmatched string) *analogdb.PostExtraction {
	return &analogdb.PostExtraction{
		PostID:           postID,
		ExtractorVersion: "2026-10-a",
		Model:            "google/gemini-2.5-flash-lite",
		Input:            "title: test",
		InputHash:        "abc",
		Raw:              json.RawMessage(`{"cameras": [], "films": [], "lenses": []}`),
		Unmatched:        json.RawMessage(unmatched),
	}
}

func TestExtractionService(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()
	defer mustClose(t, db)

	service := NewExtractionService(db)
	ctx := context.Background()

	written, skipped, err := service.UpsertExtractions(ctx, []*analogdb.PostExtraction{
		testExtraction(1, `[{"kind": "camera", "raw": "Nikon FM", "key": "nikonfm"}]`),
		testExtraction(2, `[]`),
		testExtraction(3, ``),
		testExtraction(9999, `[]`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if written != 3 {
		t.Errorf("want 3 written, got %d", written)
	}
	if len(skipped) != 1 || skipped[0] != 9999 {
		t.Errorf("want post 9999 skipped, got %v", skipped)
	}

	t.Run("upsert replaces", func(t *testing.T) {
		updated := testExtraction(2, `[{"kind": "film", "raw": "Kodacolor 200", "key": "kodakkodacolor200"}]`)
		updated.Model = "other-model"
		if _, _, err := service.UpsertExtractions(ctx, []*analogdb.PostExtraction{updated}); err != nil {
			t.Fatal(err)
		}
		got, err := service.FindExtractions(ctx, &analogdb.ExtractionFilter{Limit: 10, Full: true})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 3 {
			t.Fatalf("want 3 extractions, got %d", len(got))
		}
		if got[1].PostID != 2 || got[1].Model != "other-model" {
			t.Errorf("want post 2 updated, got %+v", got[1])
		}
		if !got[1].Updated.After(got[1].Created) && !got[1].Updated.Equal(got[1].Created) {
			t.Errorf("want updated at or after created, got %v %v", got[1].Updated, got[1].Created)
		}
	})

	t.Run("full includes input and raw", func(t *testing.T) {
		got, err := service.FindExtractions(ctx, &analogdb.ExtractionFilter{Limit: 1, Full: true})
		if err != nil {
			t.Fatal(err)
		}
		if got[0].Input != "title: test" || len(got[0].Raw) == 0 {
			t.Errorf("want input and raw, got %+v", got[0])
		}
		got, err = service.FindExtractions(ctx, &analogdb.ExtractionFilter{Limit: 1})
		if err != nil {
			t.Fatal(err)
		}
		if got[0].Input != "" || got[0].Raw != nil {
			t.Errorf("want no input or raw, got %+v", got[0])
		}
	})

	t.Run("filters", func(t *testing.T) {
		ids := func(filter *analogdb.ExtractionFilter) []int {
			t.Helper()
			filter.Limit = 10
			got, err := service.FindExtractions(ctx, filter)
			if err != nil {
				t.Fatal(err)
			}
			out := []int{}
			for _, e := range got {
				out = append(out, e.PostID)
			}
			return out
		}
		yes, no := true, false
		camera, film, key := "camera", "film", "nikonfm"
		before := 3
		cases := []struct {
			name   string
			filter *analogdb.ExtractionFilter
			want   []int
		}{
			{"all newest first", &analogdb.ExtractionFilter{}, []int{3, 2, 1}},
			{"has unmatched", &analogdb.ExtractionFilter{HasUnmatched: &yes}, []int{2, 1}},
			{"no unmatched", &analogdb.ExtractionFilter{HasUnmatched: &no}, []int{3}},
			{"kind camera", &analogdb.ExtractionFilter{Kind: &camera}, []int{1}},
			{"kind film", &analogdb.ExtractionFilter{Kind: &film}, []int{2}},
			{"key", &analogdb.ExtractionFilter{Key: &key}, []int{1}},
			{"before id", &analogdb.ExtractionFilter{BeforeID: &before}, []int{2, 1}},
		}
		for _, c := range cases {
			got := ids(c.filter)
			if len(got) != len(c.want) {
				t.Errorf("%s: want %v, got %v", c.name, c.want, got)
				continue
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("%s: want %v, got %v", c.name, c.want, got)
					break
				}
			}
		}
	})

	t.Run("deleted post removes extraction", func(t *testing.T) {
		if err := NewPostService(db).DeletePost(ctx, 3); err != nil {
			t.Fatal(err)
		}
		got, err := service.FindExtractions(ctx, &analogdb.ExtractionFilter{Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Errorf("want 2 extractions after delete, got %d", len(got))
		}
	})
}

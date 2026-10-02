package postgres

import (
	"context"
	"slices"
	"testing"
)

func TestScrapeService_CaptionMissingPostIDs(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	service := NewScrapeService(db)
	ctx := context.Background()

	if _, err := db.db.Exec(`INSERT INTO post_captions (post_id, caption, model, version, raw) VALUES
		(1, 'a sunset', 'm', 'v1', '{}'),
		(2, NULL, 'm', 'v2', '{}')`); err != nil {
		t.Fatal(err)
	}

	v1, v2 := "v1", "v2"
	cases := []struct {
		name    string
		version *string
		want    []int
	}{
		{"no version", nil, []int{3}},
		{"version v1", &v1, []int{2, 3}},
		{"version v2", &v2, []int{1, 3}},
	}
	for _, c := range cases {
		got, err := service.CaptionMissingPostIDs(ctx, c.version)
		if err != nil {
			t.Fatalf("%s: CaptionMissingPostIDs failed: %v", c.name, err)
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: want %v, got %v", c.name, c.want, got)
		}
	}
}

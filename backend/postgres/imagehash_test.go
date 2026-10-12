package postgres

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/evanofslack/analogdb"
)

// base has the top bit set, so it is stored as a negative bigint
const base = "c3a1f0e0d8b8b0f0"

// flip returns base with its lowest n bits flipped
func flip(t *testing.T, n int) int64 {
	t.Helper()
	h, err := analogdb.ParseImageHash(base)
	if err != nil {
		t.Fatal(err)
	}
	return h ^ (1<<n - 1)
}

func TestScrapeService_ImageHashes(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	service := NewScrapeService(db)
	ctx := context.Background()

	missingIDs := func() []int {
		t.Helper()
		posts, err := service.ImageHashMissingPosts(ctx)
		if err != nil {
			t.Fatalf("ImageHashMissingPosts failed: %v", err)
		}
		ids := []int{}
		for _, p := range posts {
			if p.URL == "" {
				t.Errorf("want low url for post %d", p.ID)
			}
			ids = append(ids, p.ID)
		}
		return ids
	}
	if got := missingIDs(); !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("want all seed posts missing, got %v", got)
	}

	var before time.Time
	var updatesBefore int
	if err := db.db.QueryRow(`SELECT updated, (SELECT count(*) FROM post_updates) FROM pictures WHERE id = 1`).Scan(&before, &updatesBefore); err != nil {
		t.Fatal(err)
	}

	n, err := service.SetImageHashes(ctx, []analogdb.ImageHash{
		{ID: 1, Hash: base},
		{ID: 2, Hash: "0000000000000000"},
		{ID: 999, Hash: base},
	})
	if err != nil {
		t.Fatalf("SetImageHashes failed: %v", err)
	}
	if n != 2 {
		t.Errorf("want 2 posts written, got %d", n)
	}
	if got := missingIDs(); !slices.Equal(got, []int{3}) {
		t.Errorf("want only post 3 missing, got %v", got)
	}

	var after time.Time
	var stored int64
	var updates int
	if err := db.db.QueryRow(`SELECT updated, image_hash FROM pictures WHERE id = 1`).Scan(&after, &stored); err != nil {
		t.Fatal(err)
	}
	if !after.Equal(before) {
		t.Errorf("want updated unchanged, was %v, now %v", before, after)
	}
	if got := analogdb.FormatImageHash(stored); got != base {
		t.Errorf("want stored hash %s, got %s", base, got)
	}
	if err := db.db.QueryRow(`SELECT count(*) FROM post_updates`).Scan(&updates); err != nil {
		t.Fatal(err)
	}
	if updates != updatesBefore {
		t.Errorf("want no new post_updates rows, had %d, now %d", updatesBefore, updates)
	}

	if _, err := service.SetImageHashes(ctx, []analogdb.ImageHash{{ID: 3, Hash: "bad"}}); analogdb.ErrorCode(err) != analogdb.ERRBADREQUEST {
		t.Errorf("want bad request for a bad hash, got %v", err)
	}
}

func TestScrapeService_FindImageMatch(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	service := NewScrapeService(db)
	ctx := context.Background()

	if _, err := db.db.Exec(`UPDATE pictures SET image_hash = $1, greyscale = false WHERE id = 2`, flip(t, 0)); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		hash      int64
		grayscale bool
		found     bool
	}{
		{"same hash", flip(t, 0), false, true},
		{"3 bits off", flip(t, 3), false, true},
		{"4 bits off", flip(t, 4), false, true},
		{"5 bits off", flip(t, 5), false, false},
		{"grayscale differs", flip(t, 0), true, false},
	}
	for _, tt := range tests {
		id, found, err := service.FindImageMatch(ctx, tt.hash, tt.grayscale, 4)
		if err != nil {
			t.Fatalf("%s: FindImageMatch failed: %v", tt.name, err)
		}
		if found != tt.found || (found && id != 2) {
			t.Errorf("%s: want found %v on post 2, got %v on post %d", tt.name, tt.found, found, id)
		}
	}
}

func TestScrapeService_DuplicatePosts(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	service := NewScrapeService(db)
	ctx := context.Background()

	dupes, err := service.DuplicatePosts(ctx, 4)
	if err != nil {
		t.Fatalf("DuplicatePosts failed: %v", err)
	}
	if len(dupes) != 0 {
		t.Fatalf("want no duplicates without hashes, got %v", dupes)
	}

	// post 2 is the oldest copy, 1 and 3 are reposts at the same time
	db.SetImageBase("https://img.test/")
	if _, err := db.db.Exec(`UPDATE pictures SET greyscale = false, image_hash = $1, time = 100, lowurl = 'low2.jpg' WHERE id = 2`, flip(t, 0)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.db.Exec(`UPDATE pictures SET greyscale = false, image_hash = $1, time = 200 WHERE id = 1`, flip(t, 2)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.db.Exec(`UPDATE pictures SET greyscale = false, image_hash = $1, time = 200 WHERE id = 3`, flip(t, 0)); err != nil {
		t.Fatal(err)
	}

	dupes, err = service.DuplicatePosts(ctx, 4)
	if err != nil {
		t.Fatalf("DuplicatePosts failed: %v", err)
	}
	want := []analogdb.DuplicatePost{
		{KeepID: 2, DuplicateID: 1, Distance: 2, KeepURL: "https://img.test/low2.jpg", DuplicateURL: "https://example.com/low1.jpg"},
		{KeepID: 2, DuplicateID: 3, Distance: 0, KeepURL: "https://img.test/low2.jpg", DuplicateURL: "https://example.com/low3.jpg"},
	}
	if !slices.Equal(dupes, want) {
		t.Errorf("want %+v, got %+v", want, dupes)
	}

	// post 1 is 2 bits off, outside a 1 bit cutoff
	dupes, err = service.DuplicatePosts(ctx, 1)
	if err != nil {
		t.Fatalf("DuplicatePosts failed: %v", err)
	}
	if len(dupes) != 1 || dupes[0].DuplicateID != 3 || dupes[0].DuplicateURL != "https://example.com/low3.jpg" {
		t.Errorf("want only post 3 within 1 bit, got %+v", dupes)
	}

	// a black and white copy is not a duplicate of the colour original
	if _, err := db.db.Exec(`UPDATE pictures SET greyscale = true WHERE id = 3`); err != nil {
		t.Fatal(err)
	}
	dupes, err = service.DuplicatePosts(ctx, 4)
	if err != nil {
		t.Fatalf("DuplicatePosts failed: %v", err)
	}
	if len(dupes) != 1 || dupes[0].DuplicateID != 1 {
		t.Errorf("want only post 1 as a duplicate, got %+v", dupes)
	}
}

func TestPostService_CreatePostImageHash(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	service := NewPostService(db)
	ctx := context.Background()

	create := func(permalink string, hash *string) error {
		_, err := service.CreatePost(ctx, &analogdb.CreatePost{
			Title:     "hashed",
			Author:    "u/test",
			Permalink: permalink,
			Time:      1642000000,
			Images: []analogdb.Image{
				{Label: "low", Url: "http://example.com/" + permalink + "/low.jpg"},
				{Label: "medium", Url: "http://example.com/" + permalink + "/med.jpg"},
				{Label: "high", Url: "http://example.com/" + permalink + "/high.jpg"},
				{Label: "raw", Url: "http://example.com/" + permalink + "/raw.jpg"},
			},
			Colors:    []analogdb.Color{{Hex: "#FF0000", Css: "red", Html: "red", Percent: 1}},
			ImageHash: hash,
		})
		return err
	}

	hash := base
	if err := create("hashed_post", &hash); err != nil {
		t.Fatalf("CreatePost failed: %v", err)
	}
	var stored int64
	if err := db.db.QueryRow(`SELECT image_hash FROM pictures WHERE permalink = 'hashed_post'`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if got := analogdb.FormatImageHash(stored); got != base {
		t.Errorf("want stored hash %s, got %s", base, got)
	}

	bad := "nothex"
	if err := create("bad_hash_post", &bad); analogdb.ErrorCode(err) != analogdb.ERRBADREQUEST {
		t.Errorf("want bad request for a bad hash, got %v", err)
	}
}

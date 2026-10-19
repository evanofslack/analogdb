package postgres

import (
	"context"
	"fmt"
	"slices"
	"testing"

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

// near returns base with two bits flipped, so near posts are 2 bits from base
// and 4 bits from each other
func near(t *testing.T, i int) int64 {
	return flip(t, 0) ^ (3 << (8 + 2*i))
}

type hashedPost struct {
	author        string
	width, height int
	time          int
	hash          int64
	grayscale     bool
}

func createHashedPost(t *testing.T, db *DB, p hashedPost) int {
	t.Helper()
	permalink := fmt.Sprintf("hashed_%d_%d", p.time, p.hash)
	hash := analogdb.FormatImageHash(p.hash)
	post, err := NewPostService(db).CreatePost(context.Background(), &analogdb.CreatePost{
		Title:     "hashed",
		Author:    p.author,
		Permalink: permalink,
		Time:      p.time,
		Grayscale: p.grayscale,
		Images: []analogdb.Image{
			{Label: "low", Url: "http://example.com/" + permalink + "/low.jpg", Width: p.width / 10, Height: p.height / 10},
			{Label: "medium", Url: "http://example.com/" + permalink + "/med.jpg"},
			{Label: "high", Url: "http://example.com/" + permalink + "/high.jpg"},
			{Label: "raw", Url: "http://example.com/" + permalink + "/raw.jpg", Width: p.width, Height: p.height},
		},
		Colors:    []analogdb.Color{{Hex: "#FF0000", Css: "red", Html: "red", Percent: 1}},
		ImageHash: &hash,
	})
	if err != nil {
		t.Fatal(err)
	}
	return post.Id
}

func TestScrapeService_FindImageMatch(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	service := NewScrapeService(db)
	ctx := context.Background()

	keep := createHashedPost(t, db, hashedPost{author: "u/a", width: 2000, height: 3000, time: 100, hash: flip(t, 0)})

	tests := []struct {
		name  string
		query analogdb.ImageMatchQuery
		found bool
	}{
		{"exact, other author", analogdb.ImageMatchQuery{Hash: flip(t, 0), Author: "u/b", Width: 1000, Height: 1500}, true},
		{"2 bits, other author", analogdb.ImageMatchQuery{Hash: flip(t, 2), Author: "u/b", Width: 1000, Height: 1500}, true},
		{"3 bits, other author", analogdb.ImageMatchQuery{Hash: flip(t, 3), Author: "u/b", Width: 1000, Height: 1500}, false},
		{"4 bits, same author", analogdb.ImageMatchQuery{Hash: flip(t, 4), Author: "u/a", Width: 1000, Height: 1500}, true},
		{"5 bits, same author", analogdb.ImageMatchQuery{Hash: flip(t, 5), Author: "u/a", Width: 1000, Height: 1500}, false},
		{"other aspect", analogdb.ImageMatchQuery{Hash: flip(t, 0), Author: "u/a", Width: 1000, Height: 1610}, false},
		{"grayscale differs", analogdb.ImageMatchQuery{Hash: flip(t, 0), Grayscale: true, Author: "u/a", Width: 1000, Height: 1500}, false},
	}
	for _, tt := range tests {
		match, found, err := service.FindImageMatch(ctx, tt.query)
		if err != nil {
			t.Fatalf("%s: FindImageMatch failed: %v", tt.name, err)
		}
		if found != tt.found || (found && (match.ID != keep || match.SameAuthor != (tt.query.Author == "u/a"))) {
			t.Errorf("%s: want found %v on post %d, got %v on %+v", tt.name, tt.found, keep, found, match)
		}
	}

	// a plain image lands near many posts, so an exact match is ignored
	for i := range analogdb.DuplicateCrowdSize {
		createHashedPost(t, db, hashedPost{author: "u/other", width: 2000, height: 3000, time: 200 + i, hash: near(t, i)})
	}
	_, found, err := service.FindImageMatch(ctx, tests[0].query)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("want no match on a crowded hash")
	}
}

func TestScrapeService_DuplicatePosts(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	service := NewScrapeService(db)
	ctx := context.Background()

	dupes, err := service.DuplicatePosts(ctx)
	if err != nil {
		t.Fatalf("DuplicatePosts failed: %v", err)
	}
	if len(dupes) != 0 {
		t.Fatalf("want no duplicates without hashes, got %v", dupes)
	}

	keep := createHashedPost(t, db, hashedPost{author: "u/a", width: 2000, height: 3000, time: 100, hash: flip(t, 0)})
	repost := createHashedPost(t, db, hashedPost{author: "u/b", width: 1000, height: 1500, time: 200, hash: flip(t, 0)})
	edit := createHashedPost(t, db, hashedPost{author: "u/a", width: 2000, height: 3000, time: 300, hash: flip(t, 4)})
	createHashedPost(t, db, hashedPost{author: "u/c", width: 2000, height: 3000, time: 400, hash: flip(t, 0) ^ 7<<40})
	createHashedPost(t, db, hashedPost{author: "u/a", width: 3000, height: 2000, time: 500, hash: flip(t, 0)})
	createHashedPost(t, db, hashedPost{author: "u/a", width: 2000, height: 3000, time: 600, hash: flip(t, 0), grayscale: true})

	dupes, err = service.DuplicatePosts(ctx)
	if err != nil {
		t.Fatalf("DuplicatePosts failed: %v", err)
	}
	keepURL := fmt.Sprintf("http://example.com/hashed_100_%d/low.jpg", flip(t, 0))
	want := []analogdb.DuplicatePost{
		{KeepID: keep, DuplicateID: repost, Distance: 0, SameAuthor: false, KeepURL: keepURL, DuplicateURL: fmt.Sprintf("http://example.com/hashed_200_%d/low.jpg", flip(t, 0))},
		{KeepID: keep, DuplicateID: edit, Distance: 4, SameAuthor: true, KeepURL: keepURL, DuplicateURL: fmt.Sprintf("http://example.com/hashed_300_%d/low.jpg", flip(t, 4))},
	}
	if !slices.Equal(dupes, want) {
		t.Errorf("want %+v, got %+v", want, dupes)
	}

	// once the original is crowded by plain images it keeps nothing
	for i := range analogdb.DuplicateCrowdSize {
		createHashedPost(t, db, hashedPost{author: "u/other", width: 2000, height: 3000, time: 1000 + i, hash: near(t, i)})
	}
	dupes, err = service.DuplicatePosts(ctx)
	if err != nil {
		t.Fatalf("DuplicatePosts failed: %v", err)
	}
	if len(dupes) != 0 {
		t.Errorf("want no duplicates of a crowded post, got %+v", dupes)
	}
}

func TestScrapeService_ReportDuplicates(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	service := NewScrapeService(db)
	ctx := context.Background()

	dupes := []analogdb.DuplicatePost{{KeepID: 1, DuplicateID: 2, Distance: 0}, {KeepID: 1, DuplicateID: 99999, Distance: 1}}
	reported, err := service.ReportDuplicates(ctx, dupes)
	if err != nil {
		t.Fatalf("ReportDuplicates failed: %v", err)
	}
	if reported != 1 {
		t.Fatalf("want 1 report for the live post, got %d", reported)
	}
	var reason, message string
	var duplicateOf int
	if err := db.db.QueryRow(`SELECT reason, message, duplicate_of FROM post_reports WHERE post_id = 2`).Scan(&reason, &message, &duplicateOf); err != nil {
		t.Fatal(err)
	}
	if reason != "duplicate" || message != "duplicate of post 1, 0 bits apart, other author" || duplicateOf != 1 {
		t.Errorf("unexpected report %q %q duplicate of %d", reason, message, duplicateOf)
	}

	if reported, err = service.ReportDuplicates(ctx, dupes); err != nil || reported != 0 {
		t.Errorf("want no second open report, got %d, %v", reported, err)
	}

	if _, err := db.db.Exec(`UPDATE post_reports SET resolved_at = NOW() WHERE post_id = 2`); err != nil {
		t.Fatal(err)
	}
	if reported, err = service.ReportDuplicates(ctx, dupes); err != nil || reported != 1 {
		t.Errorf("want a new report once the old one is resolved, got %d, %v", reported, err)
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

package postgres

import (
	"context"
	"slices"
	"testing"

	"github.com/evanofslack/analogdb"
)

func TestKeywordService_GetKeywordSummaryDays(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()
	defer mustClose(t, db)
	ctx := context.Background()

	if _, err := db.db.Exec(`UPDATE pictures SET created = now() - interval '2 days' WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.db.Exec(`UPDATE pictures SET created = now() - interval '30 days' WHERE id IN (2, 3)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.db.Exec(`INSERT INTO keywords (word, weight, post_id) VALUES ('portrait', 0.5, 1)`); err != nil {
		t.Fatal(err)
	}

	service := NewKeywordService(db)
	limit, days := 50, 7
	recent, err := service.GetKeywordSummary(ctx, &analogdb.KeywordFilter{Limit: &limit, Days: &days})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for _, kw := range *recent {
		got[kw.Word] = kw.Count
	}
	want := map[string]int{"sunset": 1, "landscape": 1, "golden hour": 1, "portrait": 1}
	if len(got) != len(want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	for word, count := range want {
		if got[word] != count {
			t.Errorf("want %s=%d, got %v", word, count, got)
		}
	}

	all, err := service.GetKeywordSummary(ctx, &analogdb.KeywordFilter{Limit: &limit})
	if err != nil {
		t.Fatal(err)
	}
	if len(*all) != 9 || (*all)[0].Word != "portrait" || (*all)[0].Count != 2 {
		t.Errorf("unexpected summary %+v", *all)
	}
}

func TestKeywordService_TagCounts(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()
	defer mustClose(t, db)

	if _, err := db.db.Exec(`INSERT INTO keywords (word, weight, post_id) VALUES ('portrait', 0.5, 1), ('portrait', 0.4, 3)`); err != nil {
		t.Fatal(err)
	}

	counts, total, err := NewKeywordService(db).TagCounts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Errorf("want 3 tagged posts, got %d", total)
	}
	if counts["portrait"] != 2 || counts["sunset"] != 1 || len(counts) != 9 {
		t.Errorf("unexpected counts %v", counts)
	}
}

// seedKeywordPosts adds posts 4 to 7 with scores 10, 40, 30, 20, post 5 is nsfw
func seedKeywordPosts(t *testing.T, db *DB) {
	t.Helper()
	for i, score := range []int{10, 40, 30, 20} {
		id := i + 4
		if _, err := db.db.Exec(`
			INSERT INTO pictures (id, url, title, author, permalink, score, nsfw, greyscale, time, width, height, sprocket,
				lowurl, lowwidth, lowheight, medurl, medwidth, medheight, highurl, highwidth, highheight)
			VALUES ($1::int, 'raw' || $1::text, 'post ' || $1::text, 'u/a', 'link' || $1::text, $2, $3, false, 1641168000, 1, 1, false,
				'low' || $1::text, 1, 1, 'med' || $1::text, 1, 1, 'high' || $1::text, 1, 1)`, id, score, id == 5); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.db.Exec(`
		INSERT INTO keywords (word, weight, post_id) VALUES
		('beach', 0.9, 4), ('beach', 0.9, 5), ('beach', 0.9, 6), ('beach', 0.9, 7),
		('ocean', 0.8, 4), ('ocean', 0.8, 5), ('ocean', 0.8, 6),
		('new york', 0.7, 4), ('new york', 0.7, 7),
		('self-portrait', 0.6, 6), ('sunset', 0.5, 4)`); err != nil {
		t.Fatal(err)
	}
}

func topPostIDs(posts []analogdb.CatalogPost) []int {
	ids := []int{}
	for _, p := range posts {
		ids = append(ids, p.Id)
	}
	return ids
}

func TestKeywordService_GetKeywordSummaryTopPosts(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()
	defer mustClose(t, db)
	seedKeywordPosts(t, db)
	ctx := context.Background()
	service := NewKeywordService(db)

	limit, top, minCount := 50, 2, 2
	summary, err := service.GetKeywordSummary(ctx, &analogdb.KeywordFilter{Limit: &limit, TopPosts: &top, MinCount: &minCount})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]analogdb.KeywordSummary{}
	for _, kw := range *summary {
		got[kw.Word] = kw
	}
	if len(got) != 4 || got["sunset"].Count != 2 || got["beach"].Count != 4 || got["ocean"].Count != 3 || got["new york"].Count != 2 {
		t.Fatalf("unexpected summary %+v", *summary)
	}
	if ids := topPostIDs(got["beach"].TopPosts); !slices.Equal(ids, []int{6, 7}) {
		t.Errorf("want beach covers [6 7] without the nsfw post, got %v", ids)
	}
	if ids := topPostIDs(got["new york"].TopPosts); !slices.Equal(ids, []int{7, 4}) {
		t.Errorf("want new york covers [7 4], got %v", ids)
	}
	if p := got["beach"].TopPosts[0]; p.Score != 30 || p.Title != "post 6" || len(p.Images) != 2 || p.Images[0].Url != "low6" {
		t.Errorf("unexpected cover %+v", p)
	}

	plain, err := service.GetKeywordSummary(ctx, &analogdb.KeywordFilter{Limit: &limit})
	if err != nil {
		t.Fatal(err)
	}
	if len(*plain) != 13 || (*plain)[0].Word != "beach" || (*plain)[0].TopPosts != nil {
		t.Errorf("unexpected summary without new params %+v", *plain)
	}

	if _, err := db.db.Exec(`UPDATE pictures SET created = now() - interval '30 days'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.db.Exec(`UPDATE pictures SET created = now() - interval '2 days' WHERE id IN (4, 5)`); err != nil {
		t.Fatal(err)
	}
	days := 7
	top = 1
	recent, err := service.GetKeywordSummary(ctx, &analogdb.KeywordFilter{Limit: &limit, Days: &days, TopPosts: &top, MinCount: &minCount})
	if err != nil {
		t.Fatal(err)
	}
	if len(*recent) != 2 {
		t.Fatalf("want beach and ocean in the last week, got %+v", *recent)
	}
	for _, kw := range *recent {
		if kw.Count != 2 {
			t.Errorf("want recent count 2, got %+v", kw)
		}
		if ids := topPostIDs(kw.TopPosts); !slices.Equal(ids, []int{6}) {
			t.Errorf("want all time cover [6] for %s, got %v", kw.Word, ids)
		}
	}
}

func TestKeywordService_FindKeyword(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()
	defer mustClose(t, db)
	seedKeywordPosts(t, db)
	ctx := context.Background()
	service := NewKeywordService(db)

	detail, err := service.FindKeyword(ctx, "beach", 6)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Word != "beach" || detail.Count != 4 {
		t.Errorf("unexpected detail %+v", detail)
	}
	if ids := topPostIDs(detail.TopPosts); !slices.Equal(ids, []int{6, 7, 4}) {
		t.Errorf("want covers [6 7 4], got %v", ids)
	}

	_, total, err := NewPostService(db).FindPosts(ctx, &analogdb.PostFilter{Keywords: &[]string{"beach"}})
	if err != nil {
		t.Fatal(err)
	}
	if total != detail.Count {
		t.Errorf("want posts total to match the keyword count %d, got %d", detail.Count, total)
	}

	for _, word := range []string{"new york", "self-portrait"} {
		detail, err := service.FindKeyword(ctx, word, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(detail.TopPosts) != 1 {
			t.Errorf("want 1 cover for %s, got %+v", word, detail)
		}
	}

	_, err = service.FindKeyword(ctx, "zzzz", 6)
	if analogdb.ErrorCode(err) != analogdb.ERRNOTFOUND {
		t.Errorf("want not found, got %v", err)
	}
}

func TestKeywordService_CoOccurring(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()
	defer mustClose(t, db)
	seedKeywordPosts(t, db)

	shared, err := NewKeywordService(db).CoOccurring(context.Background(), "beach")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"ocean": 3, "new york": 2, "self-portrait": 1, "sunset": 1}
	if len(shared) != len(want) {
		t.Fatalf("want %v, got %v", want, shared)
	}
	for word, n := range want {
		if shared[word] != n {
			t.Errorf("want %s=%d, got %v", word, n, shared)
		}
	}
}

package postgres

import (
	"context"
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

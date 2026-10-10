package postgres

import (
	"context"
	"testing"

	"github.com/evanofslack/analogdb"
)

func TestDeletePostWritesTombstone(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()
	defer mustClose(t, db)

	posts := NewPostService(db)
	ctx := context.Background()

	var permalink, url, lowURL, medURL, highURL string
	var postTime int64
	if err := db.db.QueryRow(`SELECT permalink, url, lowurl, medurl, highurl, time FROM pictures WHERE id = 1`).
		Scan(&permalink, &url, &lowURL, &medURL, &highURL, &postTime); err != nil {
		t.Fatal(err)
	}

	if err := posts.DeletePost(ctx, 1, string(analogdb.ReportTakedown)); err != nil {
		t.Fatal(err)
	}

	var gotPermalink, gotURL, gotLow, gotMed, gotHigh, reason string
	var postedAt int64
	if err := db.db.QueryRow(`
		SELECT permalink, url, low_url, med_url, high_url, extract(epoch FROM posted_at)::bigint, reason
		FROM removed_posts WHERE post_id = 1`).
		Scan(&gotPermalink, &gotURL, &gotLow, &gotMed, &gotHigh, &postedAt, &reason); err != nil {
		t.Fatal(err)
	}
	if gotPermalink != permalink || gotURL != url || gotLow != lowURL || gotMed != medURL || gotHigh != highURL {
		t.Errorf("tombstone urls don't match the post")
	}
	if postedAt != postTime {
		t.Errorf("want posted_at %d, got %d", postTime, postedAt)
	}
	if reason != string(analogdb.ReportTakedown) {
		t.Errorf("want reason takedown, got %q", reason)
	}

	t.Run("empty reason is null", func(t *testing.T) {
		if err := posts.DeletePost(ctx, 2, ""); err != nil {
			t.Fatal(err)
		}
		var isNull bool
		if err := db.db.QueryRow(`SELECT reason IS NULL FROM removed_posts WHERE post_id = 2`).Scan(&isNull); err != nil {
			t.Fatal(err)
		}
		if !isNull {
			t.Error("want null reason")
		}
	})

	t.Run("removed permalink can't be created again", func(t *testing.T) {
		_, err := posts.CreatePost(ctx, &analogdb.CreatePost{
			Title:     "back again",
			Author:    "u/someone",
			Permalink: permalink,
			Time:      1642000000,
			Images: []analogdb.Image{
				{Label: "low", Url: "http://example.com/new-low.jpg", Width: 200, Height: 300},
				{Label: "medium", Url: "http://example.com/new-med.jpg", Width: 600, Height: 900},
				{Label: "high", Url: "http://example.com/new-high.jpg", Width: 1200, Height: 1800},
				{Label: "raw", Url: "http://example.com/new-raw.jpg", Width: 2400, Height: 3600},
			},
			Colors: []analogdb.Color{{Hex: "#FF0000", Css: "rgb(255,0,0)", Html: "red", Percent: 1}},
		})
		if code := analogdb.ErrorCode(err); code != analogdb.ERRCONFLICT {
			t.Errorf("want conflict, got %v", err)
		}
	})
}

func TestReportService(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()
	defer mustClose(t, db)

	reports := NewReportService(db)
	posts := NewPostService(db)
	ctx := context.Background()

	create := func(postID int, reason analogdb.ReportReason, email string) int {
		t.Helper()
		id, err := reports.CreateReport(ctx, &analogdb.CreateReport{PostID: postID, Reason: reason, Message: "hello", Email: email})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}

	first := create(1, analogdb.ReportTakedown, "me@example.com")
	second := create(1, analogdb.ReportNsfwMislabeled, "")
	third := create(2, analogdb.ReportNotFilm, "")

	t.Run("missing post", func(t *testing.T) {
		_, err := reports.CreateReport(ctx, &analogdb.CreateReport{PostID: 9999, Reason: analogdb.ReportOther})
		if code := analogdb.ErrorCode(err); code != analogdb.ERRNOTFOUND {
			t.Errorf("want not found, got %v", err)
		}
	})

	t.Run("find newest first with post snapshot", func(t *testing.T) {
		got, err := reports.FindReports(ctx, &analogdb.ReportFilter{Status: analogdb.ReportStatusOpen, Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 3 || got[0].ID != third || got[2].ID != first {
			t.Fatalf("want reports %d..%d newest first, got %+v", third, first, got)
		}
		if got[2].Email == nil || *got[2].Email != "me@example.com" {
			t.Errorf("want email on first report")
		}
		if got[1].Email != nil {
			t.Errorf("want no email on second report")
		}
		if got[0].Post.Permalink == nil || got[0].Post.Removed {
			t.Errorf("want a live post snapshot, got %+v", got[0].Post)
		}

		page, err := reports.FindReports(ctx, &analogdb.ReportFilter{Status: analogdb.ReportStatusAll, BeforeID: &second, Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if len(page) != 1 || page[0].ID != first {
			t.Errorf("want only report %d before %d, got %d reports", first, second, len(page))
		}
	})

	t.Run("resolve drops email", func(t *testing.T) {
		if err := reports.ResolveReport(ctx, first); err != nil {
			t.Fatal(err)
		}
		if err := reports.ResolveReport(ctx, first); err != nil {
			t.Errorf("want resolving twice to be fine, got %v", err)
		}
		got, err := reports.FindReportByID(ctx, first)
		if err != nil {
			t.Fatal(err)
		}
		if got.ResolvedAt == nil || got.Email != nil {
			t.Errorf("want resolved with no email, got %+v", got)
		}
		if err := reports.ResolveReport(ctx, 9999); analogdb.ErrorCode(err) != analogdb.ERRNOTFOUND {
			t.Errorf("want not found, got %v", err)
		}

		open, _ := reports.FindReports(ctx, &analogdb.ReportFilter{Status: analogdb.ReportStatusOpen, Limit: 10})
		resolved, _ := reports.FindReports(ctx, &analogdb.ReportFilter{Status: analogdb.ReportStatusResolved, Limit: 10})
		if len(open) != 2 || len(resolved) != 1 {
			t.Errorf("want 2 open and 1 resolved, got %d and %d", len(open), len(resolved))
		}
	})

	t.Run("takedown keeps the snapshot", func(t *testing.T) {
		if err := posts.DeletePost(ctx, 1, string(analogdb.ReportTakedown)); err != nil {
			t.Fatal(err)
		}
		if err := reports.ResolvePostReports(ctx, 1); err != nil {
			t.Fatal(err)
		}
		got, err := reports.FindReportByID(ctx, second)
		if err != nil {
			t.Fatal(err)
		}
		if !got.Post.Removed || got.Post.Permalink == nil || got.ResolvedAt == nil {
			t.Errorf("want a resolved report on a removed post, got %+v", got)
		}
		other, err := reports.FindReportByID(ctx, third)
		if err != nil {
			t.Fatal(err)
		}
		if other.ResolvedAt != nil {
			t.Error("want reports on other posts left open")
		}

		permalinks, err := reports.RemovedPermalinks(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(permalinks) != 1 || permalinks[0] != *got.Post.Permalink {
			t.Errorf("want the removed permalink, got %v", permalinks)
		}
	})
}

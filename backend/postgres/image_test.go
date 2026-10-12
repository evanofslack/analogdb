package postgres

import (
	"context"
	"testing"

	"github.com/evanofslack/analogdb"
)

func TestImageURL(t *testing.T) {
	cases := []struct {
		stored string
		want   string
	}{
		{"abc-low.jpeg", "https://img.test/abc-low.jpeg"},
		{"https://d3i73ktnzbi69i.cloudfront.net/abc-low.jpeg", "https://d3i73ktnzbi69i.cloudfront.net/abc-low.jpeg"},
		{"https://example.com/abc.jpg", "https://example.com/abc.jpg"},
		{"", ""},
	}
	for _, c := range cases {
		if got := imageURL("https://img.test", c.stored); got != c.want {
			t.Errorf("imageURL(%q) = %q, want %q", c.stored, got, c.want)
		}
	}
}

func TestImageKey(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"https://d3i73ktnzbi69i.cloudfront.net/abc-raw.jpeg", "abc-raw.jpeg"},
		{"https://i.analogdb.com/abc-raw.jpeg", "abc-raw.jpeg"},
		{"https://img.test/abc-raw.jpeg", "abc-raw.jpeg"},
		{"https://example.com/abc.jpg", "https://example.com/abc.jpg"},
		{"https://i.analogdb.com/", "https://i.analogdb.com/"},
		{"abc-raw.jpeg", "abc-raw.jpeg"},
		{"", ""},
	}
	for _, c := range cases {
		if got := imageKey("https://img.test", c.url); got != c.want {
			t.Errorf("imageKey(%q) = %q, want %q", c.url, got, c.want)
		}
	}
}

func TestImageKeysStoredAndServed(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()
	defer mustClose(t, db)

	db.SetImageBase("https://img.test/")
	posts := NewPostService(db)
	reports := NewReportService(db)
	ctx := context.Background()

	const cdn = "https://d3i73ktnzbi69i.cloudfront.net/"
	post, err := posts.CreatePost(ctx, &analogdb.CreatePost{
		Title:     "Keyed Post",
		Author:    "u/testuser",
		Permalink: "keyed_post",
		Time:      1642000000,
		Images: []analogdb.Image{
			{Label: "low", Url: cdn + "k-low.jpeg", Width: 200, Height: 300},
			{Label: "medium", Url: "https://i.analogdb.com/k-medium.jpeg", Width: 600, Height: 900},
			{Label: "high", Url: cdn + "k-high.jpeg", Width: 1200, Height: 1800},
			{Label: "raw", Url: cdn + "k-raw.jpeg", Width: 2400, Height: 3600},
		},
		Colors: []analogdb.Color{{Hex: "#FF0000", Css: "red", Html: "red", Percent: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}

	var url, low, med, high string
	if err := db.db.QueryRow(`SELECT url, lowurl, medurl, highurl FROM pictures WHERE id = $1`, post.Id).
		Scan(&url, &low, &med, &high); err != nil {
		t.Fatal(err)
	}
	if url != "k-raw.jpeg" || low != "k-low.jpeg" || med != "k-medium.jpeg" || high != "k-high.jpeg" {
		t.Errorf("want keys stored, got %q %q %q %q", url, low, med, high)
	}

	found, err := posts.FindPostByID(ctx, post.Id)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"https://img.test/k-low.jpeg", "https://img.test/k-medium.jpeg", "https://img.test/k-high.jpeg", "https://img.test/k-raw.jpeg"}
	for i, image := range found.Images {
		if image.Url != want[i] {
			t.Errorf("image %s: want %q, got %q", image.Label, want[i], image.Url)
		}
	}

	id, err := reports.CreateReport(ctx, &analogdb.CreateReport{PostID: post.Id, Reason: analogdb.ReportTakedown})
	if err != nil {
		t.Fatal(err)
	}
	if err := posts.DeletePost(ctx, post.Id, string(analogdb.ReportTakedown)); err != nil {
		t.Fatal(err)
	}
	var removedLow string
	if err := db.db.QueryRow(`SELECT low_url FROM removed_posts WHERE post_id = $1`, post.Id).Scan(&removedLow); err != nil {
		t.Fatal(err)
	}
	if removedLow != "k-low.jpeg" {
		t.Errorf("want removed low_url key, got %q", removedLow)
	}
	report, err := reports.FindReportByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if report.Post.LowURL == nil || *report.Post.LowURL != "https://img.test/k-low.jpeg" {
		t.Errorf("want report low_url on the image base, got %v", report.Post.LowURL)
	}
}

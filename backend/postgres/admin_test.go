package postgres

import (
	"context"
	"testing"

	"github.com/evanofslack/analogdb"
)

func TestAdminStats(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()
	defer mustClose(t, db)

	stats, err := NewAdminService(db).Stats(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	wantCounts := analogdb.AdminCounts{Posts: 3, Authors: 3, Cameras: 3, Films: 3, Keywords: 9}
	if stats.Counts != wantCounts {
		t.Errorf("want counts %+v, got %+v", wantCounts, stats.Counts)
	}
	if stats.Posted != (analogdb.AdminPostedCounts{}) {
		t.Errorf("want no recent posts, got %+v", stats.Posted)
	}
	if f := stats.Freshness; f.NewestPost == nil || *f.NewestPost != 1641168000 {
		t.Errorf("want newest post 1641168000, got %v", f.NewestPost)
	}
	if f := stats.Freshness; f.ScoreUpdate == nil || *f.ScoreUpdate != 1641427200 {
		t.Errorf("want score update 1641427200, got %v", f.ScoreUpdate)
	}
	if stats.Database.Bytes <= 0 {
		t.Error("want database size")
	}
	if got := len(stats.Database.Tables); got != len(adminTables) {
		t.Errorf("want %d tables, got %d", len(adminTables), got)
	}
	if stats.Database.MigrationVersion <= 0 || stats.Database.MigrationDirty {
		t.Errorf("want clean migration version, got %d dirty=%v", stats.Database.MigrationVersion, stats.Database.MigrationDirty)
	}
}

func TestAdminQuality(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()
	defer mustClose(t, db)

	quality, err := NewAdminService(db).Quality(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	wantCoverage := analogdb.AdminCoverage{Total: 3, Camera: 3, Film: 3, Description: 2, Keywords: 3, Colors: 3, FocalLength: 3, Aperture: 3}
	if quality.Coverage != wantCoverage {
		t.Errorf("want coverage %+v, got %+v", wantCoverage, quality.Coverage)
	}

	if len(quality.UnmatchedCameras) != 1 {
		t.Fatalf("want 1 unmatched camera, got %+v", quality.UnmatchedCameras)
	}
	if c := quality.UnmatchedCameras[0]; c.CameraMake != "pentax" || c.CameraModel != "k1000" || c.PostCount != 1 || c.SamplePostID != 3 {
		t.Errorf("unexpected unmatched camera %+v", c)
	}

	if len(quality.UnmatchedFilms) != 3 {
		t.Fatalf("want 3 unmatched films, got %+v", quality.UnmatchedFilms)
	}
	for _, f := range quality.UnmatchedFilms {
		if f.FilmSpeed == nil || *f.FilmSpeed != 400 {
			t.Errorf("want film speed 400 for %+v", f)
		}
	}
}

func TestAdminMissingPosts(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()
	defer mustClose(t, db)
	ctx := context.Background()
	s := NewAdminService(db)

	if _, err := db.db.ExecContext(ctx, `UPDATE pictures SET camera_make = NULL WHERE id = 1; DELETE FROM keywords WHERE post_id IN (1, 3)`); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		field    analogdb.MissingField
		beforeID *int
		want     []int
	}{
		{field: analogdb.MissingDescription, want: []int{2}},
		{field: analogdb.MissingCamera, want: []int{1}},
		{field: analogdb.MissingFilm, want: []int{}},
		{field: analogdb.MissingKeywords, want: []int{3, 1}},
		{field: analogdb.MissingKeywords, beforeID: intPtr(3), want: []int{1}},
		{field: analogdb.MissingColors, want: []int{}},
	}
	for _, tt := range tests {
		posts, err := s.MissingPosts(ctx, &analogdb.MissingPostsFilter{Field: tt.field, Limit: 10, BeforeID: tt.beforeID})
		if err != nil {
			t.Fatal(err)
		}
		got := make([]int, 0, len(posts))
		for _, p := range posts {
			got = append(got, p.ID)
		}
		if len(got) != len(tt.want) {
			t.Errorf("%s: want %v, got %v", tt.field, tt.want, got)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("%s: want %v, got %v", tt.field, tt.want, got)
				break
			}
		}
	}

	if _, err := s.MissingPosts(ctx, &analogdb.MissingPostsFilter{Field: "title", Limit: 10}); analogdb.ErrorCode(err) != analogdb.ERRBADREQUEST {
		t.Errorf("want bad request for invalid field, got %v", err)
	}
}

func intPtr(i int) *int { return &i }

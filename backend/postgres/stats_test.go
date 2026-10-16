package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/evanofslack/analogdb"
)

func unitVector(i int) []float32 {
	v := make([]float32, analogdb.StatsVectorDims)
	v[i] = 1
	return v
}

func testSeed(version string, themes ...string) *analogdb.StatsSeed {
	seed := &analogdb.StatsSeed{Version: version, Model: "ViT-B-16-laion2b_s34b_b88k"}
	for i, key := range themes {
		seed.Themes = append(seed.Themes, analogdb.StatsTheme{Key: key, Name: "theme " + key, Center: unitVector(i)})
	}
	for i, key := range analogdb.StatsPromptKeys {
		seed.Prompts = append(seed.Prompts, analogdb.StatsPrompt{Key: key, Label: key, Text: "a photo", Vector: unitVector(100 + i)})
	}
	return seed
}

func TestStatsSeed(t *testing.T) {
	db, cleanup := mustOpen(t)
	defer cleanup()
	defer mustClose(t, db)

	stats := NewStatsService(db)
	ctx := context.Background()

	if _, err := stats.FindSeed(ctx); analogdb.ErrorCode(err) != analogdb.ERRNOTFOUND {
		t.Fatalf("want not found before any seed, got %v", err)
	}

	border := analogdb.StatsRoleBorder
	first := testSeed("v1", "t00", "t01", "t02")
	first.Themes[1].Hidden = true
	first.Themes[2].Role = &border
	if err := stats.ReplaceSeed(ctx, first); err != nil {
		t.Fatal(err)
	}

	got, err := stats.FindSeed(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "v1" || got.Model != first.Model {
		t.Errorf("want v1 and model, got %q %q", got.Version, got.Model)
	}
	if len(got.Themes) != 3 || len(got.Prompts) != len(analogdb.StatsPromptKeys) {
		t.Fatalf("want 3 themes and %d prompts, got %d and %d", len(analogdb.StatsPromptKeys), len(got.Themes), len(got.Prompts))
	}
	if !got.Themes[1].Hidden || got.Themes[0].Hidden {
		t.Error("want only t01 hidden")
	}
	if got.Themes[2].Role == nil || *got.Themes[2].Role != border || got.Themes[0].Role != nil {
		t.Error("want only t02 with the border role")
	}
	for i, theme := range got.Themes {
		if len(theme.Center) != analogdb.StatsVectorDims || theme.Center[i] != 1 {
			t.Errorf("theme %s center didn't round trip", theme.Key)
		}
	}
	for _, p := range got.Prompts {
		if len(p.Vector) != analogdb.StatsVectorDims {
			t.Errorf("prompt %s vector has %d values", p.Key, len(p.Vector))
		}
	}

	t.Run("replace removes old rows", func(t *testing.T) {
		if err := stats.ReplaceSeed(ctx, testSeed("v2", "t05", "t06")); err != nil {
			t.Fatal(err)
		}
		got, err := stats.FindSeed(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got.Version != "v2" {
			t.Errorf("want v2, got %q", got.Version)
		}
		if len(got.Themes) != 2 || got.Themes[0].Key != "t05" || got.Themes[1].Key != "t06" {
			t.Errorf("want only t05 and t06, got %+v", got.Themes)
		}
		if len(got.Prompts) != len(analogdb.StatsPromptKeys) {
			t.Errorf("want %d prompts, got %d", len(analogdb.StatsPromptKeys), len(got.Prompts))
		}
	})
}

func TestLoadStatsInput(t *testing.T) {
	db, cleanup := mustOpen(t)
	defer cleanup()
	defer mustClose(t, db)

	ctx := context.Background()
	since := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.db.ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO films (film_make, film_type, film_speed, color_type, description) VALUES ('kodak', 'portra 400', 400, 'color', '')`)
	insert := `INSERT INTO pictures (id, url, permalink, author, score, time, film_make, film_type, lowurl, medurl)
		VALUES ($1::integer, 'u' || $1::text, 'p' || $1::text, $2, $3, $4, $5, $6, 'low.jpg', 'med.jpg')`
	exec(insert, 1, "a", 10, since.Add(-time.Hour).Unix(), "kodak", "portra 400")
	exec(insert, 2, "a", 10, since.Add(time.Hour).Unix(), "kodak", "portra 400")
	exec(insert, 3, "b", 20, since.Add(2*time.Hour).Unix(), "kodak", "unknown")
	exec(`INSERT INTO colors (hex, css, html, percent, post_id, oklab_l, oklab_a, oklab_b, chroma, hue)
		VALUES ('ff0000', 'red', 'red', 0.5, 1, 0.6, 0.2, 0.1, 0.22, 26), ('ff0000', 'red', 'red', 0.5, 2, 0.6, 0.2, 0.1, 0.22, 26)`)
	exec(`INSERT INTO keywords (word, weight, post_id) VALUES ('neon', 0.9, 1), ('neon', 0.9, 2), ('neon', 0.5, 2), ('tree', 0.9, 2)`)

	in, err := NewStatsService(db).LoadStatsInput(ctx, since, []string{"neon"})
	if err != nil {
		t.Fatal(err)
	}
	if len(in.Posts) != 2 || in.Posts[0].ID != 2 || in.Posts[1].ID != 3 {
		t.Fatalf("want posts 2 and 3 in the window, got %+v", in.Posts)
	}
	if p := in.Posts[0]; p.FilmSlug != "kodak-portra-400" || p.FilmColorType != "color" || len(p.Images) != 2 {
		t.Errorf("want post 2 joined to the catalog with images, got %+v", p)
	}
	if p := in.Posts[1]; p.FilmSlug != "" {
		t.Errorf("want no film for a type outside the catalog, got %q", p.FilmSlug)
	}
	if in.Posts[0].Popularity != 0 || in.Posts[1].Popularity != 1 {
		t.Errorf("want popularity ranked within the month, got %v and %v", in.Posts[0].Popularity, in.Posts[1].Popularity)
	}
	if len(in.Colors) != 1 || in.Colors[0].PostID != 2 {
		t.Errorf("want only post 2's color, got %+v", in.Colors)
	}
	if len(in.Keywords) != 1 || in.Keywords[0].Word != "neon" || in.Keywords[0].PostID != 2 {
		t.Errorf("want one neon keyword for post 2, got %+v", in.Keywords)
	}
}

func TestStatsSnapshots(t *testing.T) {
	db, cleanup := mustOpen(t)
	defer cleanup()
	defer mustClose(t, db)

	stats := NewStatsService(db)
	ctx := context.Background()

	if _, err := stats.FindLatestSnapshot(ctx); analogdb.ErrorCode(err) != analogdb.ERRNOTFOUND {
		t.Fatalf("want not found before any snapshot, got %v", err)
	}
	for i := range statsSnapshotsKept + 2 {
		snapshot := &analogdb.StatsSnapshot{SeedVersion: "v1", BuildMs: i, Body: []byte(fmt.Sprintf(`{"version": 1, "build": %d}`, i))}
		if err := stats.CreateSnapshot(ctx, snapshot); err != nil {
			t.Fatal(err)
		}
	}

	snapshots, err := stats.FindSnapshots(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != statsSnapshotsKept {
		t.Fatalf("want %d snapshots kept, got %d", statsSnapshotsKept, len(snapshots))
	}
	if snapshots[0].BuildMs != statsSnapshotsKept+1 || snapshots[0].Size == 0 {
		t.Errorf("want the newest first with a size, got %+v", snapshots[0])
	}

	latest, err := stats.FindLatestSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var body struct{ Build int }
	if err := json.Unmarshal(latest.Body, &body); err != nil {
		t.Fatal(err)
	}
	if body.Build != statsSnapshotsKept+1 || latest.SeedVersion != "v1" {
		t.Errorf("want the newest body, got %+v", body)
	}
}

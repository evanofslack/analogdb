package postgres

import (
	"context"
	"testing"

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

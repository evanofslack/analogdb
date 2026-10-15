package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/colorspace"
	"github.com/evanofslack/analogdb/postgres"
)

func TestRederiveColors(t *testing.T) {
	db, raw := mustOpenPagingDB(t)
	s := mustOpen(t)
	t.Cleanup(func() { mustClose(t, s) })
	s.config.Auth.AdminUsername, s.config.Auth.AdminPassword = "admin", "secret"
	s.config.Auth.ScraperUsername, s.config.Auth.ScraperPassword = "scraper", "scrape"
	service := postgres.NewPostService(db)
	s.PostService = service
	s.ColorService = postgres.NewColorService(db)
	ctx := context.Background()

	palettes := map[int][]colorspace.Swatch{}
	n := len(pagingHexes)
	for id := 1; id <= pagingPosts; id++ {
		colors := []analogdb.Color{
			{Hex: pagingHexes[id%n], Percent: 0.5},
			{Hex: pagingHexes[(id/n)%n], Percent: 0.3},
			{Hex: pagingHexes[(id*5+3)%n], Percent: 0.2},
		}
		if err := service.PatchPost(ctx, &analogdb.PatchPost{Colors: &colors}, id); err != nil {
			t.Fatal(err)
		}
		for _, c := range colors {
			palettes[id] = append(palettes[id], colorspace.Swatch{Hex: c.Hex, Percent: c.Percent})
		}
	}
	if _, err := raw.ExecContext(ctx, "UPDATE colors SET html = 'red', chroma = NULL"); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, "UPDATE pictures SET color_hue = NULL, color_strength = NULL, rainbow_key = NULL"); err != nil {
		t.Fatal(err)
	}

	request := func(user, password string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/v1/admin/colors/rederive", nil)
		if user != "" {
			r.SetBasicAuth(user, password)
		}
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, r)
		return w
	}
	if w := request("", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", w.Code)
	}
	if w := request("scraper", "scrape"); w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for the scraper, got %d", w.Code)
	}
	w := request("admin", "secret")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result analogdb.ColorRederive
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Posts != pagingPosts || result.ColorsUpdated != 3*pagingPosts || result.PostsUpdated == 0 {
		t.Errorf("unexpected result %+v", result)
	}

	rows, err := raw.QueryContext(ctx, "SELECT id, color_strength, rainbow_key FROM pictures")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	keys := 0
	for rows.Next() {
		var id int
		var strength, key sql.NullFloat64
		if err := rows.Scan(&id, &strength, &key); err != nil {
			t.Fatal(err)
		}
		c, ok, err := colorspace.PalettePostColor(palettes[id], false)
		if err != nil {
			t.Fatal(err)
		}
		if key.Valid != ok || ok && (key.Float64 != colorspace.RainbowKey(id, c) || float32(strength.Float64) != float32(c.Strength)) {
			t.Errorf("post %d: expected key %v strength %v, got %v %v", id, colorspace.RainbowKey(id, c), c.Strength, key, strength)
		}
		if key.Valid {
			keys++
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if keys == 0 {
		t.Error("expected rainbow keys")
	}

	var html string
	if err := raw.QueryRowContext(ctx, "SELECT html FROM colors WHERE hex = '#008080' LIMIT 1").Scan(&html); err != nil || html != colorspace.Teal {
		t.Errorf("expected teal restored, got %q %v", html, err)
	}
}

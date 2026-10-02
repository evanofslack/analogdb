package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/evanofslack/analogdb"
)

type catalogPicture struct {
	title string
	score int
	nsfw  bool
}

// mustInsertCatalogPictures adds kodak tri-x pictures shot on a leica m6
func mustInsertCatalogPictures(t *testing.T, db *DB, pictures []catalogPicture) {
	t.Helper()
	for i, p := range pictures {
		_, err := db.db.Exec(`
			INSERT INTO pictures (
				url, title, author, permalink, score, nsfw, greyscale, time, width, height, sprocket,
				lowurl, lowwidth, lowheight, medurl, medwidth, medheight, highurl, highwidth, highheight,
				camera_make, camera_model, film_make, film_type, film_speed
			) VALUES (
				$1, $2, 'u/catalog', $3, $4, $5, true, 1641254400, 3000, 2000, false,
				$6, 300, 200, $7, 800, 533, 'https://example.com/high.jpg', 1600, 1067,
				'leica', 'm6', 'kodak', 'tri-x', 400
			)`,
			fmt.Sprintf("https://example.com/catalog%d.jpg", i),
			p.title,
			fmt.Sprintf("reddit.com/catalog_%d", i),
			p.score,
			p.nsfw,
			fmt.Sprintf("https://example.com/low_catalog%d.jpg", i),
			fmt.Sprintf("https://example.com/med_catalog%d.jpg", i),
		)
		if err != nil {
			t.Fatalf("Insert catalog picture, err=%v", err)
		}
	}
}

var testCatalogPictures = []catalogPicture{
	{title: "low", score: 10},
	{title: "best", score: 50},
	{title: "hidden", score: 100, nsfw: true},
	{title: "middle", score: 30},
}

func checkTopPosts(t *testing.T, top []analogdb.CatalogPost, wantTitles []string) {
	t.Helper()
	if len(top) != len(wantTitles) {
		t.Fatalf("Expected %d top posts, got %d", len(wantTitles), len(top))
	}
	for i, p := range top {
		if p.Title != wantTitles[i] {
			t.Errorf("Expected top post %d to be %q, got %q", i, wantTitles[i], p.Title)
		}
		if p.Id <= 0 {
			t.Errorf("Expected positive id, got %d", p.Id)
		}
		if len(p.Images) != 2 || p.Images[0].Label != "low" || p.Images[1].Label != "medium" {
			t.Errorf("Expected low and medium images, got %+v", p.Images)
		}
		for _, img := range p.Images {
			if img.Url == "" || img.Width == 0 || img.Height == 0 {
				t.Errorf("Expected full image, got %+v", img)
			}
		}
	}
	for i := 1; i < len(top); i++ {
		if top[i-1].Score < top[i].Score {
			t.Errorf("Top posts not ordered by score: %d < %d", top[i-1].Score, top[i].Score)
		}
	}
}

func TestFilmService_Catalog(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()
	mustInsertCatalogPictures(t, db, testCatalogPictures)

	service := NewFilmService(db)
	ctx := context.Background()

	t.Run("min count filters films", func(t *testing.T) {
		minCount := 4
		filter := &analogdb.FilmFilter{MinCount: &minCount}
		films, err := service.FindFilms(ctx, filter)
		if err != nil {
			t.Fatalf("Films failed: %v", err)
		}
		if len(films) != 1 {
			t.Fatalf("Expected 1 film, got %d", len(films))
		}
		if films[0].Make != "kodak" || films[0].Type != "tri-x" {
			t.Errorf("Expected kodak tri-x, got %s %s", films[0].Make, films[0].Type)
		}
		if films[0].PostCount != 4 {
			t.Errorf("Expected post count 4 including nsfw, got %d", films[0].PostCount)
		}
	})

	t.Run("min count above all counts", func(t *testing.T) {
		minCount := 5
		filter := &analogdb.FilmFilter{MinCount: &minCount}
		films, err := service.FindFilms(ctx, filter)
		if err != nil {
			t.Fatalf("Films failed: %v", err)
		}
		if len(films) != 0 {
			t.Errorf("Expected 0 films, got %d", len(films))
		}
	})

	t.Run("min count with other filters", func(t *testing.T) {
		minCount := 1
		make := "kodak"
		speed := 400
		filter := &analogdb.FilmFilter{Make: &make, Speed: &speed, MinCount: &minCount}
		films, err := service.FindFilms(ctx, filter)
		if err != nil {
			t.Fatalf("Films failed: %v", err)
		}
		if len(films) != 1 {
			t.Errorf("Expected 1 film, got %d", len(films))
		}
	})

	t.Run("top posts ordered by score without nsfw", func(t *testing.T) {
		top := 10
		filter := &analogdb.FilmFilter{TopPosts: &top}
		films, err := service.FindFilms(ctx, filter)
		if err != nil {
			t.Fatalf("Films failed: %v", err)
		}
		for _, f := range films {
			if f.Make == "kodak" && f.Type == "tri-x" {
				checkTopPosts(t, f.TopPosts, []string{"best", "middle", "low"})
			} else if len(f.TopPosts) != 0 {
				t.Errorf("Expected no top posts for %s %s, got %d", f.Make, f.Type, len(f.TopPosts))
			}
		}
	})

	t.Run("top posts count capped", func(t *testing.T) {
		top := 2
		minCount := 1
		filter := &analogdb.FilmFilter{TopPosts: &top, MinCount: &minCount}
		films, err := service.FindFilms(ctx, filter)
		if err != nil {
			t.Fatalf("Films failed: %v", err)
		}
		if len(films) != 1 {
			t.Fatalf("Expected 1 film, got %d", len(films))
		}
		checkTopPosts(t, films[0].TopPosts, []string{"best", "middle"})
	})

	t.Run("no top posts by default", func(t *testing.T) {
		films, err := service.FindFilms(ctx, &analogdb.FilmFilter{})
		if err != nil {
			t.Fatalf("Films failed: %v", err)
		}
		for _, f := range films {
			if f.TopPosts != nil {
				t.Errorf("Expected nil top posts, got %d", len(f.TopPosts))
			}
		}
	})
}

func TestCameraService_Catalog(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()
	mustInsertCatalogPictures(t, db, testCatalogPictures)

	service := NewCameraService(db)
	ctx := context.Background()

	t.Run("min count filters cameras", func(t *testing.T) {
		minCount := 2
		filter := &analogdb.CameraFilter{MinCount: &minCount}
		cameras, err := service.FindCameras(ctx, filter)
		if err != nil {
			t.Fatalf("Cameras failed: %v", err)
		}
		if len(cameras) != 1 {
			t.Fatalf("Expected 1 camera, got %d", len(cameras))
		}
		if cameras[0].Make != "leica" || cameras[0].Model != "m6" {
			t.Errorf("Expected leica m6, got %s %s", cameras[0].Make, cameras[0].Model)
		}
		if cameras[0].PostCount != 4 {
			t.Errorf("Expected post count 4 including nsfw, got %d", cameras[0].PostCount)
		}
	})

	t.Run("min count of one", func(t *testing.T) {
		minCount := 1
		filter := &analogdb.CameraFilter{MinCount: &minCount}
		cameras, err := service.FindCameras(ctx, filter)
		if err != nil {
			t.Fatalf("Cameras failed: %v", err)
		}
		if len(cameras) != 3 {
			t.Errorf("Expected 3 cameras, got %d", len(cameras))
		}
	})

	t.Run("top posts ordered by score without nsfw", func(t *testing.T) {
		top := 6
		make := "leica"
		filter := &analogdb.CameraFilter{Make: &make, TopPosts: &top}
		cameras, err := service.FindCameras(ctx, filter)
		if err != nil {
			t.Fatalf("Cameras failed: %v", err)
		}
		if len(cameras) != 1 {
			t.Fatalf("Expected 1 camera, got %d", len(cameras))
		}
		checkTopPosts(t, cameras[0].TopPosts, []string{"best", "middle", "low"})
	})

	t.Run("top posts count capped", func(t *testing.T) {
		top := 1
		minCount := 2
		filter := &analogdb.CameraFilter{TopPosts: &top, MinCount: &minCount}
		cameras, err := service.FindCameras(ctx, filter)
		if err != nil {
			t.Fatalf("Cameras failed: %v", err)
		}
		if len(cameras) != 1 {
			t.Fatalf("Expected 1 camera, got %d", len(cameras))
		}
		checkTopPosts(t, cameras[0].TopPosts, []string{"best"})
	})
}

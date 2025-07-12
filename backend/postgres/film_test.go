package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/evanofslack/analogdb"
)

func TestFilmService_FindFilms(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	service := NewFilmService(db)
	ctx := context.Background()

	t.Run("find all films", func(t *testing.T) {
		filter := analogdb.NewFilmFilter(nil, nil, nil, nil, nil, nil, nil, nil, nil)
		films, err := service.FindFilms(ctx, filter)
		if err != nil {
			t.Fatalf("Films failed: %v", err)
		}

		expectedFilms := 3
		if len(films) != expectedFilms {
			t.Errorf("Expected %d films, got %d", expectedFilms, len(films))
		}
	})

	t.Run("verify film ordering", func(t *testing.T) {
		filter := analogdb.NewFilmFilter(nil, nil, nil, nil, nil, nil, nil, nil, nil)
		films, err := service.FindFilms(ctx, filter)
		if err != nil {
			t.Fatalf("Films failed: %v", err)
		}

		if len(films) < 2 {
			t.Skip("Need at least 2 films to test ordering")
		}

		for i := 0; i < len(films)-1; i++ {
			current := films[i]
			next := films[i+1]

			if current.Make > next.Make {
				t.Errorf("Films not ordered by make: %q > %q", current.Make, next.Make)
			} else if current.Make == next.Make {
				if current.Type > next.Type {
					t.Errorf("Films not ordered by type: %q > %q", current.Type, next.Type)
				} else if current.Type == next.Type && current.Speed > next.Speed {
					t.Errorf("Films not ordered by speed: %d > %d", current.Speed, next.Speed)
				}
			}
		}
	})

	t.Run("verify film struct fields", func(t *testing.T) {
		filter := analogdb.NewFilmFilter(nil, nil, nil, nil, nil, nil, nil, nil, nil)
		films, err := service.FindFilms(ctx, filter)
		if err != nil {
			t.Fatalf("Films failed: %v", err)
		}

		for i, film := range films {
			if film.Make == "" {
				t.Errorf("Film at index %d has empty Make", i)
			}
			if film.Id <= 0 {
				t.Errorf("Film at index %d has invalid ID: %d", i, film.Id)
			}
			if film.Type == "" {
				t.Errorf("Film at index %d has empty Type", i)
			}
			if film.Speed <= 0 {
				t.Errorf("Film at index %d has invalid Speed: %d", i, film.Speed)
			}
			if film.ColorType == "" {
				t.Errorf("Film at index %d has empty ColorType", i)
			}
			if film.Created.IsZero() {
				t.Errorf("Film at index %d has zero Created timestamp", i)
			}
			if film.Updated.IsZero() {
				t.Errorf("Film at index %d has zero Updated timestamp", i)
			}
		}
	})

	t.Run("no duplicate films", func(t *testing.T) {
		filter := analogdb.NewFilmFilter(nil, nil, nil, nil, nil, nil, nil, nil, nil)
		films, err := service.FindFilms(ctx, filter)
		if err != nil {
			t.Fatalf("Films failed: %v", err)
		}

		seen := make(map[string]bool)
		for _, film := range films {
			key := fmt.Sprintf("%s-%s-%d", film.Make, film.Type, film.Speed)
			if seen[key] {
				t.Errorf("Duplicate film found: key=%s make=%s type=%s speed=%d (total_films=%d)", key, film.Make, film.Type, film.Speed, len(films))
			}
			seen[key] = true
		}
	})
}

func TestFilmService_CreateFilm(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	service := NewFilmService(db)
	ctx := context.Background()

	t.Run("create new film", func(t *testing.T) {
		film := &analogdb.CreateFilm{
			Make:        "ilford",
			Type:        "hp5",
			Speed:       400,
			ColorType:   "bw",
			Description: "High speed black and white film",
		}

		created, err := service.CreateFilm(ctx, film)
		if err != nil {
			t.Fatalf("CreateFilm failed: %v", err)
		}

		if created.Id <= 0 {
			t.Errorf("Expected positive ID, got %d", created.Id)
		}
		if created.Make != film.Make {
			t.Errorf("Expected Make %q, got %q", film.Make, created.Make)
		}
		if created.Type != film.Type {
			t.Errorf("Expected Type %q, got %q", film.Type, created.Type)
		}
		if created.Speed != film.Speed {
			t.Errorf("Expected Speed %d, got %d", film.Speed, created.Speed)
		}
		if created.ColorType != film.ColorType {
			t.Errorf("Expected ColorType %q, got %q", film.ColorType, created.ColorType)
		}
		if created.Description != film.Description {
			t.Errorf("Expected Description %q, got %q", film.Description, created.Description)
		}
	})

	t.Run("create film increases count", func(t *testing.T) {
		filter := analogdb.NewFilmFilter(nil, nil, nil, nil, nil, nil, nil, nil, nil)
		initialFilms, err := service.FindFilms(ctx, filter)
		if err != nil {
			t.Fatalf("Films failed: %v", err)
		}
		initialCount := len(initialFilms)

		film := &analogdb.CreateFilm{
			Make:        "rollei",
			Type:        "infrared",
			Speed:       400,
			ColorType:   "bw",
			Description: "Infrared sensitive film",
		}

		_, err = service.CreateFilm(ctx, film)
		if err != nil {
			t.Fatalf("CreateFilm failed: %v", err)
		}

		filmsAfterCreate, err := service.FindFilms(ctx, filter)
		if err != nil {
			t.Fatalf("Films failed: %v", err)
		}

		expectedCount := initialCount + 1
		if len(filmsAfterCreate) != expectedCount {
			t.Errorf("Expected %d films after creation, got %d", expectedCount, len(filmsAfterCreate))
		}
	})

	t.Run("create duplicate film with conflict", func(t *testing.T) {
		film := &analogdb.CreateFilm{
			Make:        "kodak",
			Type:        "tri-x",
			Speed:       400,
			ColorType:   "bw",
			Description: "Duplicate film",
		}

		_, err := service.CreateFilm(ctx, film)
		if err != nil {
			t.Error("Expected no error when creating duplicate film, just updated fields")
		}
	})
}

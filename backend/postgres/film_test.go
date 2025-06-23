package postgres

import (
	"fmt"
	"context"
	"testing"
)

func TestFilmService(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	service := NewFilmService(db)
	ctx := context.Background()

	t.Run("find all films", func(t *testing.T) {
		films, err := service.Films(ctx)
		if err != nil {
			t.Fatalf("Films failed: %v", err)
		}

		expectedCount := 3
		if len(films) != expectedCount {
			t.Errorf("Expected %d films, got %d", expectedCount, len(films))
		}
	})

	t.Run("verify film ordering", func(t *testing.T) {
		films, err := service.Films(ctx)
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
		films, err := service.Films(ctx)
		if err != nil {
			t.Fatalf("Films failed: %v", err)
		}

		for i, film := range films {
			if film.Id <= 0 {
				t.Errorf("Film at index %d has invalid ID: %d", i, film.Id)
			}
			if film.Make == "" {
				t.Errorf("Film at index %d has empty Make", i)
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
		films, err := service.Films(ctx)
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

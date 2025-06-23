package postgres

import (
	"context"
	"fmt"
	"testing"

)

func TestCameraService(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	service := NewCameraService(db)
	ctx := context.Background()

	t.Run("find all cameras", func(t *testing.T) {
		cameras, err := service.Cameras(ctx)
		if err != nil {
			t.Fatalf("Cameras failed: %v", err)
		}

		expectedCount := 3
		if len(cameras) != expectedCount {
			t.Errorf("Expected %d cameras, got %d", expectedCount, len(cameras))
		}
	})

	t.Run("verify camera ordering", func(t *testing.T) {
		cameras, err := service.Cameras(ctx)
		if err != nil {
			t.Fatalf("Cameras failed: %v", err)
		}

		if len(cameras) < 2 {
			t.Skip("Need at least 2 cameras to test ordering")
		}

		for i := 0; i < len(cameras)-1; i++ {
			current := cameras[i]
			next := cameras[i+1]

			if current.Make > next.Make {
				t.Errorf("Cameras not ordered by make: %q > %q", current.Make, next.Make)
			} else if current.Make == next.Make && current.Model > next.Model {
				t.Errorf("Cameras not ordered by model: %q > %q", current.Model, next.Model)
			}
		}
	})

	t.Run("verify camera struct fields", func(t *testing.T) {
		cameras, err := service.Cameras(ctx)
		if err != nil {
			t.Fatalf("Cameras failed: %v", err)
		}

		for i, camera := range cameras {
			if camera.Id <= 0 {
				t.Errorf("Camera at index %d has invalid ID: %d", i, camera.Id)
			}
			if camera.Make == "" {
				t.Errorf("Camera at index %d has empty Make", i)
			}
			if camera.Model == "" {
				t.Errorf("Camera at index %d has empty Model", i)
			}
			if camera.Created.IsZero() {
				t.Errorf("Camera at index %d has zero Created timestamp", i)
			}
			if camera.Updated.IsZero() {
				t.Errorf("Camera at index %d has zero Updated timestamp", i)
			}
		}
	})

	t.Run("no duplicate cameras", func(t *testing.T) {
		cameras, err := service.Cameras(ctx)
		if err != nil {
			t.Fatalf("Cameras failed: %v", err)
		}

		seen := make(map[string]bool)
		for _, camera := range cameras {
			key := fmt.Sprintf("%s-%s", camera.Make, camera.Model)
			if seen[key] {
				t.Errorf("Duplicate camera found: key=%s make=%s model=%s (total_cameras=%d)", key, camera.Make, camera.Model, len(cameras))
			}
			seen[key] = true
		}
	})
}

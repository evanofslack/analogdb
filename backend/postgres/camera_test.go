package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/evanofslack/analogdb"
)

func TestCameraService_AllCameras(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	service := NewCameraService(db)
	ctx := context.Background()

	t.Run("find all cameras", func(t *testing.T) {
		cameras, err := service.AllCameras(ctx)
		if err != nil {
			t.Fatalf("Cameras failed: %v", err)
		}

		expectedCount := 3
		if len(cameras) != expectedCount {
			t.Errorf("Expected %d cameras, got %d", expectedCount, len(cameras))
		}
	})

	t.Run("verify camera ordering", func(t *testing.T) {
		cameras, err := service.AllCameras(ctx)
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
		cameras, err := service.AllCameras(ctx)
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
		cameras, err := service.AllCameras(ctx)
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

func TestCameraService_CreateCamera(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	service := NewCameraService(db)
	ctx := context.Background()

	t.Run("create new camera", func(t *testing.T) {
		camera := &analogdb.Camera{
			Make:        "pentax",
			Model:       "k1000",
			Description: "Student camera with manual controls",
		}

		created, err := service.CreateCamera(ctx, camera)
		if err != nil {
			t.Fatalf("CreateCamera failed: %v", err)
		}

		if created.Id <= 0 {
			t.Errorf("Expected positive ID, got %d", created.Id)
		}
		if created.Make != camera.Make {
			t.Errorf("Expected Make %q, got %q", camera.Make, created.Make)
		}
		if created.Model != camera.Model {
			t.Errorf("Expected Model %q, got %q", camera.Model, created.Model)
		}
		if created.Description != camera.Description {
			t.Errorf("Expected Description %q, got %q", camera.Description, created.Description)
		}
	})

	t.Run("create camera increases count", func(t *testing.T) {
		initialCameras, err := service.AllCameras(ctx)
		if err != nil {
			t.Fatalf("Cameras failed: %v", err)
		}
		initialCount := len(initialCameras)

		camera := &analogdb.Camera{
			Make:        "olympus",
			Model:       "om-1",
			Description: "Compact professional SLR",
		}

		_, err = service.CreateCamera(ctx, camera)
		if err != nil {
			t.Fatalf("CreateCamera failed: %v", err)
		}

		camerasAfterCreate, err := service.AllCameras(ctx)
		if err != nil {
			t.Fatalf("Cameras failed: %v", err)
		}

		expectedCount := initialCount + 1
		if len(camerasAfterCreate) != expectedCount {
			t.Errorf("Expected %d cameras after creation, got %d", expectedCount, len(camerasAfterCreate))
		}
	})

	t.Run("create duplicate camera with conflict", func(t *testing.T) {
		camera := &analogdb.Camera{
			Make:        "canon",
			Model:       "ae-1",
			Description: "Duplicate camera",
		}

		_, err := service.CreateCamera(ctx, camera)
		if err == nil {
			t.Error("Expected error when creating duplicate camera, got nil")
		}
	})
}

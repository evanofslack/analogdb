package postgres

import (
	"context"
	"testing"

	"github.com/evanofslack/analogdb"
)

func TestAuthorService_FindAuthors(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	service := NewAuthorService(db)
	ctx := context.Background()

	t.Run("find all authors", func(t *testing.T) {
		authors, err := service.FindAuthors(ctx)
		if err != nil {
			t.Fatalf("FindAuthors failed: %v", err)
		}

		expectedAuthors := []string{
			"u/photographer1",
			"u/streetphotographer",
			"u/portraitist",
		}

		if len(authors) != len(expectedAuthors) {
			t.Errorf("Expected %d authors, got %d", len(expectedAuthors), len(authors))
		}

		for i, expectedAuthor := range expectedAuthors {
			if i >= len(authors) {
				t.Errorf("Missing author at index %d: expected %q", i, expectedAuthor)
				continue
			}
			if authors[i] != expectedAuthor {
				t.Errorf("Author at index %d: expected %q, got %q", i, expectedAuthor, authors[i])
			}
		}
	})

	t.Run("verify author format includes u/ prefix", func(t *testing.T) {
		authors, err := service.FindAuthors(ctx)
		if err != nil {
			t.Fatalf("FindAuthors failed: %v", err)
		}

		for i, author := range authors {
			if len(author) < 2 || author[:2] != "u/" {
				t.Errorf("Author at index %d should start with 'u/': got %q", i, author)
			}
		}
	})

	t.Run("no duplicate authors in result", func(t *testing.T) {
		authors, err := service.FindAuthors(ctx)
		if err != nil {
			t.Fatalf("FindAuthors failed: %v", err)
		}

		// Check for duplicates
		seen := make(map[string]bool)
		for _, author := range authors {
			if seen[author] {
				t.Errorf("Duplicate author found: %q", author)
			}
			seen[author] = true
		}
	})
}

func TestAuthorService_FindAuthorsAfterPostDeletion(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	postService := NewPostService(db)
	authorService := NewAuthorService(db)
	ctx := context.Background()

	// Get initial author count
	initialAuthors, err := authorService.FindAuthors(ctx)
	if err != nil {
		t.Fatalf("FindAuthors failed: %v", err)
	}
	initialCount := len(initialAuthors)

	// Delete a post
	err = postService.DeletePost(ctx, 1)
	if err != nil {
		t.Fatalf("DeletePost failed: %v", err)
	}

	// Get authors after deletion
	authorsAfterDeletion, err := authorService.FindAuthors(ctx)
	if err != nil {
		t.Fatalf("FindAuthors failed: %v", err)
	}

	expectedCount := initialCount - 1
	if len(authorsAfterDeletion) != expectedCount {
		t.Errorf("Expected %d authors after deletion, got %d", expectedCount, len(authorsAfterDeletion))
	}

	// Verify the deleted author is no longer in the list
	deletedAuthor := "u/photographer1" // This was the author for post id 1
	for _, author := range authorsAfterDeletion {
		if author == deletedAuthor {
			t.Errorf("Deleted author %q should not be in results", deletedAuthor)
		}
	}
}

func TestAuthorService_FindAuthorsAfterPostCreation(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	postService := NewPostService(db)
	authorService := NewAuthorService(db)
	ctx := context.Background()

	initialAuthors, err := authorService.FindAuthors(ctx)
	if err != nil {
		t.Fatalf("FindAuthors failed: %v", err)
	}
	initialCount := len(initialAuthors)

	// Create new post with new author
	createPost := &analogdb.CreatePost{
		Title:     "New Test Post",
		Author:    "u/newphotographer",
		Permalink: "new_test_post_unique",
		Score:     50,
		Nsfw:      false,
		Grayscale: false,
		Time:      1642000000,
		Sprocket:  false,
		Images: []analogdb.Image{
			{Label: "low", Url: "http://example.com/new_low.jpg", Width: 200, Height: 300},
			{Label: "medium", Url: "http://example.com/new_med.jpg", Width: 600, Height: 900},
			{Label: "high", Url: "http://example.com/new_high.jpg", Width: 1200, Height: 1800},
			{Label: "raw", Url: "http://example.com/new_raw.jpg", Width: 2400, Height: 3600},
		},
		Colors: []analogdb.Color{
			{Hex: "#FF0000", Css: "rgb(255,0,0)", Html: "red", Percent: 0.5},
			{Hex: "#00FF00", Css: "rgb(0,255,0)", Html: "green", Percent: 0.3},
			{Hex: "#0000FF", Css: "rgb(0,0,255)", Html: "blue", Percent: 0.15},
			{Hex: "#FFFF00", Css: "rgb(255,255,0)", Html: "yellow", Percent: 0.04},
			{Hex: "#FF00FF", Css: "rgb(255,0,255)", Html: "magenta", Percent: 0.01},
		},
		Keywords: []analogdb.Keyword{
			{Word: "new", Weight: 0.9},
		},
	}

	_, err = postService.CreatePost(ctx, createPost)
	if err != nil {
		t.Fatalf("CreatePost failed: %v", err)
	}

	authorsAfterCreation, err := authorService.FindAuthors(ctx)
	if err != nil {
		t.Fatalf("FindAuthors failed: %v", err)
	}

	expectedCount := initialCount + 1
	if len(authorsAfterCreation) != expectedCount {
		t.Errorf("Expected %d authors after creation, got %d", expectedCount, len(authorsAfterCreation))
	}

	// Verify new author in list
	newAuthor := "u/newphotographer"
	found := false
	for _, author := range authorsAfterCreation {
		if author == newAuthor {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("New author %q should be in results", newAuthor)
	}
}

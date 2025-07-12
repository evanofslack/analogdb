package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/evanofslack/analogdb"
)

func TestPostService_CreatePost(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	service := NewPostService(db)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		desc := "here is description"
		createPost := &analogdb.CreatePost{
			Title:       "Test Post",
			Author:      "u/testuser",
			Permalink:   "test_post_unique",
			Description: &desc,
			Score:       100,
			Nsfw:        false,
			Grayscale:   false,
			Time:        1642000000,
			Sprocket:    false,
			Images: []analogdb.Image{
				{Label: "low", Url: "http://example.com/low.jpg", Width: 200, Height: 300},
				{Label: "medium", Url: "http://example.com/med.jpg", Width: 600, Height: 900},
				{Label: "high", Url: "http://example.com/high.jpg", Width: 1200, Height: 1800},
				{Label: "raw", Url: "http://example.com/raw.jpg", Width: 2400, Height: 3600},
			},
			Colors: []analogdb.Color{
				{Hex: "#FF0000", Css: "rgb(255,0,0)", Html: "red", Percent: 0.5},
				{Hex: "#00FF00", Css: "rgb(0,255,0)", Html: "green", Percent: 0.3},
				{Hex: "#0000FF", Css: "rgb(0,0,255)", Html: "blue", Percent: 0.15},
				{Hex: "#FFFF00", Css: "rgb(255,255,0)", Html: "yellow", Percent: 0.04},
				{Hex: "#FF00FF", Css: "rgb(255,0,255)", Html: "magenta", Percent: 0.01},
			},
			Keywords: []analogdb.Keyword{
				{Word: "test", Weight: 0.9},
				{Word: "sample", Weight: 0.8},
			},
		}

		post, err := service.CreatePost(ctx, createPost)
		if err != nil {
			t.Fatalf("CreatePost failed: %v", err)
		}

		if post.Id == 0 {
			t.Error("Expected post ID to be set")
		}
		if post.Title != createPost.Title {
			t.Errorf("Expected title %q, got %q", createPost.Title, post.Title)
		}
		if post.Description != nil && *post.Description != *createPost.Description {
			t.Errorf("Expected description %q, got %q", *createPost.Description, *post.Description)
		}
		if post.Author != createPost.Author {
			t.Errorf("Expected author %q, got %q", createPost.Author, post.Author)
		}
		if len(post.Colors) != 5 {
			t.Errorf("Expected 5 colors, got %d", len(post.Colors))
		}
		if len(post.Keywords) != 2 {
			t.Errorf("Expected 2 keywords, got %d", len(post.Keywords))
		}
	})

	t.Run("creation with insufficient images", func(t *testing.T) {
		createPost := &analogdb.CreatePost{
			Title:     "Test Post",
			Author:    "u/testuser",
			Permalink: "test_post_unique_2",
			Images:    []analogdb.Image{{Label: "low", Url: "test.jpg"}}, // Only 1 image
			Colors:    make([]analogdb.Color, 5),
		}

		_, err := service.CreatePost(ctx, createPost)
		if err == nil {
			t.Error("Expected error for insufficient images")
		}
	})

	t.Run("creation with insufficient colors", func(t *testing.T) {
		createPost := &analogdb.CreatePost{
			Title:     "Test Post",
			Author:    "u/testuser",
			Permalink: "test_post_unique_3",
			Images:    make([]analogdb.Image, 4),
			Colors:    []analogdb.Color{{Hex: "#FF0000"}}, // Only 1 color
		}

		_, err := service.CreatePost(ctx, createPost)
		if err == nil {
			t.Error("Expected error for insufficient colors")
		}
	})
}

func TestPostService_FindPostByID(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	service := NewPostService(db)
	ctx := context.Background()

	t.Run("find existing post", func(t *testing.T) {
		post, err := service.FindPostByID(ctx, 1)
		if err != nil {
			t.Fatalf("FindPostByID failed: %v", err)
		}

		if post.Id != 1 {
			t.Errorf("Expected ID 1, got %d", post.Id)
		}
		if post.Title != "Sunset Photography" {
			t.Errorf("Expected title 'Sunset Photography', got %q", post.Title)
		}
		// Check that author prefix is stripped
		if post.Author != "photographer1" {
			t.Errorf("Expected author 'photographer1', got %q", post.Author)
		}
	})

	t.Run("find non-existent post", func(t *testing.T) {
		_, err := service.FindPostByID(ctx, 9999)
		if err == nil {
			t.Error("Expected error for non-existent post")
		}

		analogErr, ok := err.(*analogdb.Error)
		if !ok {
			t.Errorf("Expected analogdb.Error, got %T", err)
		} else if analogErr.Code != analogdb.ERRNOTFOUND {
			t.Errorf("Expected ERRNOTFOUND, got %s", analogErr.Code)
		}
	})
}

func TestPostService_FindPosts(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	service := NewPostService(db)
	ctx := context.Background()

	t.Run("find all posts", func(t *testing.T) {
		filter := &analogdb.PostFilter{}
		posts, count, err := service.FindPosts(ctx, filter)
		if err != nil {
			t.Fatalf("FindPosts failed: %v", err)
		}

		if len(posts) == 0 {
			t.Error("Expected to find posts")
		}
		if count == 0 {
			t.Error("Expected count > 0")
		}
	})

	t.Run("find posts with limit", func(t *testing.T) {
		limit := 1
		filter := &analogdb.PostFilter{Limit: &limit}
		posts, count, err := service.FindPosts(ctx, filter)
		if err != nil {
			t.Fatalf("FindPosts failed: %v", err)
		}

		if len(posts) != 1 {
			t.Errorf("Expected 1 post, got %d", len(posts))
		}
		if count == 0 {
			t.Error("Expected total count > 0")
		}
	})

	t.Run("find posts by author", func(t *testing.T) {
		author := "photographer1"
		filter := &analogdb.PostFilter{Author: &author}
		posts, _, err := service.FindPosts(ctx, filter)
		if err != nil {
			t.Fatalf("FindPosts failed: %v", err)
		}

		if len(posts) == 0 {
			t.Error("Expected to find posts by author")
		}
		for _, post := range posts {
			if post.Author != author {
				t.Errorf("Expected author %q, got %q", author, post.Author)
			}
		}
	})

	t.Run("find posts by nsfw flag", func(t *testing.T) {
		nsfw := false
		filter := &analogdb.PostFilter{Nsfw: &nsfw}
		posts, _, err := service.FindPosts(ctx, filter)
		if err != nil {
			t.Fatalf("FindPosts failed: %v", err)
		}

		for _, post := range posts {
			if post.Nsfw != nsfw {
				t.Errorf("Expected nsfw %t, got %t", nsfw, post.Nsfw)
			}
		}
	})

	t.Run("find posts by start time", func(t *testing.T) {
		start := 1640995210
		startUnix := time.Unix(int64(start), 0)
		filter := &analogdb.PostFilter{TimeStart: &startUnix}
		posts, _, err := service.FindPosts(ctx, filter)
		if err != nil {
			t.Fatalf("FindPosts failed: %v", err)
		}

		for _, post := range posts {
			if post.Time < start {
				t.Errorf("Expected start < %v, got %v", start, post.Time)
			}
		}
	})

	t.Run("find posts by end time", func(t *testing.T) {
		end := 1640995210
		endUnix := time.Unix(int64(end), 0)
		filter := &analogdb.PostFilter{TimeEnd: &endUnix}
		posts, _, err := service.FindPosts(ctx, filter)
		if err != nil {
			t.Fatalf("FindPosts failed: %v", err)
		}

		for _, post := range posts {
			if post.Time > end {
				t.Errorf("Expected end > %v, got %v", end, post.Time)
			}
		}
	})
}

func TestPostService_PatchPost(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	service := NewPostService(db)
	ctx := context.Background()

	t.Run("patch post score", func(t *testing.T) {
		newScore := 999
		patch := &analogdb.PatchPost{Score: &newScore}

		err := service.PatchPost(ctx, patch, 1)
		if err != nil {
			t.Fatalf("PatchPost failed: %v", err)
		}

		// Verify the change
		post, err := service.FindPostByID(ctx, 1)
		if err != nil {
			t.Fatalf("FindPostByID failed: %v", err)
		}
		if post.Score != newScore {
			t.Errorf("Expected score %d, got %d", newScore, post.Score)
		}
	})

	t.Run("patch post description", func(t *testing.T) {
		newDesc := "new description"
		patch := &analogdb.PatchPost{Description: &newDesc}

		err := service.PatchPost(ctx, patch, 1)
		if err != nil {
			t.Fatalf("PatchPost failed: %v", err)
		}

		// Verify the change
		post, err := service.FindPostByID(ctx, 1)
		if err != nil {
			t.Fatalf("FindPostByID failed: %v", err)
		}
		if post.Description != nil && *post.Description != newDesc {
			t.Errorf("Expected desc %q, got %q", newDesc, *post.Description)
		}
	})

	t.Run("patch post with new keywords", func(t *testing.T) {
		newKeywords := []analogdb.Keyword{
			{Word: "updated", Weight: 0.9},
			{Word: "keyword", Weight: 0.8},
		}
		patch := &analogdb.PatchPost{Keywords: &newKeywords}

		err := service.PatchPost(ctx, patch, 1)
		if err != nil {
			t.Fatalf("PatchPost failed: %v", err)
		}

		// Verify the change
		post, err := service.FindPostByID(ctx, 1)
		if err != nil {
			t.Fatalf("FindPostByID failed: %v", err)
		}
		if len(post.Keywords) != 2 {
			t.Errorf("Expected 2 keywords, got %d", len(post.Keywords))
		}
	})

	t.Run("patch with no fields", func(t *testing.T) {
		patch := &analogdb.PatchPost{} // Empty patch

		err := service.PatchPost(ctx, patch, 1)
		if err == nil {
			t.Error("Expected error for empty patch")
		}
	})
}

func TestPostService_DeletePost(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	service := NewPostService(db)
	ctx := context.Background()

	t.Run("delete existing post", func(t *testing.T) {
		// First verify the post exists
		_, err := service.FindPostByID(ctx, 1)
		if err != nil {
			t.Fatalf("Post should exist before deletion: %v", err)
		}

		err = service.DeletePost(ctx, 1)
		if err != nil {
			t.Fatalf("DeletePost failed: %v", err)
		}

		// Verify the post is gone
		_, err = service.FindPostByID(ctx, 1)
		if err == nil {
			t.Error("Post should not exist after deletion")
		}
	})

	t.Run("delete non-existent post", func(t *testing.T) {
		err := service.DeletePost(ctx, 9999)
		if err == nil {
			t.Error("Expected error when deleting non-existent post")
		}
	})
}

func TestPostService_AllPostIDs(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	service := NewPostService(db)
	ctx := context.Background()

	ids, err := service.AllPostIDs(ctx)
	if err != nil {
		t.Fatalf("AllPostIDs failed: %v", err)
	}

	if len(ids) == 0 {
		t.Error("Expected to find post IDs")
	}

	// Verify IDs are sorted
	for i := 1; i < len(ids); i++ {
		if ids[i-1] >= ids[i] {
			t.Error("Expected IDs to be sorted in ascending order")
		}
	}
}

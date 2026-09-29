package weaviate

import (
	"slices"
	"testing"
)

func TestSimilarPostIDs(t *testing.T) {
	pics := []pictureResponse{
		{postID: 10485}, {postID: 10485}, {postID: 7}, {postID: 12926},
		{postID: 10485}, {postID: 8}, {postID: 7}, {postID: 9}, {postID: 10},
	}

	tests := []struct {
		name     string
		limit    int
		expected []int
	}{
		{"trims to limit", 3, []int{10485, 7, 8}},
		{"limit larger than results", 10, []int{10485, 7, 8, 9, 10}},
		{"no limit", 0, []int{10485, 7, 8, 9, 10}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := similarPostIDs(pics, 12926, tt.limit)
			if !slices.Equal(got, tt.expected) {
				t.Errorf("similar ids = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFetchLimit(t *testing.T) {
	tests := []struct {
		limit    int
		expected int
	}{
		{0, 0},
		{1, 3},
		{12, 36},
		{50, 150},
		{100, 200},
		{300, 300},
	}

	for _, tt := range tests {
		if got := fetchLimit(tt.limit); got != tt.expected {
			t.Errorf("fetchLimit(%d) = %d, want %d", tt.limit, got, tt.expected)
		}
	}
}

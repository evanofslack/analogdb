package server

import (
	"reflect"
	"testing"

	"github.com/evanofslack/analogdb"
)

func TestParamJoiner(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected string
		newValue int
	}{
		{"first param", 0, "?", 1},
		{"second param", 1, "&", 2},
		{"third param", 5, "&", 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			numParams := tt.input
			result := paramJoiner(&numParams)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
			if numParams != tt.newValue {
				t.Errorf("expected numParams to be %d, got %d", tt.newValue, numParams)
			}
		})
	}
}

func TestStringToBool(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  bool
		expectErr bool
	}{
		{"true string", "true", true, false},
		{"false string", "false", false, false},
		{"1 string", "1", true, false},
		{"0 string", "0", false, false},
		{"invalid string", "invalid", false, true},
		{"empty string", "", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := stringToBool(tt.input)
			if tt.expectErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %t, got %t", tt.expected, result)
			}
		})
	}
}

func TestStringToInt(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  int
		expectErr bool
	}{
		{"positive integer", "123", 123, false},
		{"negative integer", "-123", -123, false},
		{"zero", "0", 0, false},
		{"invalid string", "abc", 0, true},
		{"float string", "12.34", 0, true},
		{"empty string", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := stringToInt(tt.input)
			if tt.expectErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestSetMeta(t *testing.T) {
	limit20 := 20
	limit2 := 2
	limit1 := 1
	sortTime := analogdb.SortTime
	sortScore := analogdb.SortScore
	sortRandom := analogdb.SortRandom
	seed42 := 42

	tests := []struct {
		name     string
		filter   *analogdb.PostFilter
		posts    []*analogdb.Post
		count    int
		expected Meta
	}{
		{
			name:   "basic meta with time sort",
			filter: &analogdb.PostFilter{Limit: &limit2, Sort: &sortTime},
			posts: []*analogdb.Post{
				{DisplayPost: analogdb.DisplayPost{Time: 1000, Score: 100}},
				{DisplayPost: analogdb.DisplayPost{Time: 2000, Score: 200}},
			},
			count: 100,
			expected: Meta{
				TotalPosts: 100,
				PageSize:   2,
				PageID:     2000,
				PageURL:    "/posts?sort=latest&page_size=2&page_id=2000",
			},
		},
		{
			name:   "meta with score sort",
			filter: &analogdb.PostFilter{Limit: &limit2, Sort: &sortScore},
			posts: []*analogdb.Post{
				{DisplayPost: analogdb.DisplayPost{Time: 1000, Score: 100}},
				{DisplayPost: analogdb.DisplayPost{Time: 2000, Score: 200}},
			},
			count: 50,
			expected: Meta{
				TotalPosts: 50,
				PageSize:   2,
				PageID:     200,
				PageURL:    "/posts?sort=top&page_size=2&page_id=200",
			},
		},
		{
			name:   "meta with random sort and seed",
			filter: &analogdb.PostFilter{Limit: &limit1, Sort: &sortRandom, Seed: &seed42},
			posts: []*analogdb.Post{
				{DisplayPost: analogdb.DisplayPost{Time: 1000, Score: 100}},
			},
			count: 10,
			expected: Meta{
				TotalPosts: 10,
				PageSize:   1,
				PageID:     1000,
				PageURL:    "/posts?sort=random&page_size=1&page_id=1000",
				Seed:       42,
			},
		},
		{
			name:   "end of pagination",
			filter: &analogdb.PostFilter{Limit: &limit20, Sort: &sortTime},
			posts: []*analogdb.Post{
				{DisplayPost: analogdb.DisplayPost{Time: 1000, Score: 100}},
			},
			count: 100,
			expected: Meta{
				TotalPosts: 100,
				PageSize:   20,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setMeta(tt.filter, tt.posts, tt.count)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

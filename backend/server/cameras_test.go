package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evanofslack/analogdb"
)

func TestParseToCameraFilter(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		checkFilter func(*analogdb.CameraFilter) bool
	}{
		{
			name:        "default filter",
			url:         "/cameras",
			expectError: false,
			checkFilter: func(f *analogdb.CameraFilter) bool {
				return f.Sort != nil && *f.Sort == defaultCamerasSort
			},
		},
		{
			name:        "sort alphabetical",
			url:         "/cameras?sort=alphabetical",
			expectError: false,
			checkFilter: func(f *analogdb.CameraFilter) bool {
				return f.Sort != nil && *f.Sort == analogdb.CameraSortAlphabetical
			},
		},
		{
			name:        "sort by counts",
			url:         "/cameras?sort=counts",
			expectError: false,
			checkFilter: func(f *analogdb.CameraFilter) bool {
				return f.Sort != nil && *f.Sort == analogdb.CameraSortCounts
			},
		},
		{
			name:        "invalid sort",
			url:         "/cameras?sort=invalid",
			expectError: true,
		},
		{
			name:        "page size",
			url:         "/cameras?page_size=10",
			expectError: false,
			checkFilter: func(f *analogdb.CameraFilter) bool {
				return f.Limit != nil && *f.Limit == 10
			},
		},
		{
			name:        "make filter",
			url:         "/cameras?make=Canon",
			expectError: false,
			checkFilter: func(f *analogdb.CameraFilter) bool {
				return f.Make != nil && *f.Make == "Canon"
			},
		},
		{
			name:        "model filter",
			url:         "/cameras?model=AE-1",
			expectError: false,
			checkFilter: func(f *analogdb.CameraFilter) bool {
				return f.Model != nil && *f.Model == "AE-1"
			},
		},
		{
			name:        "id filter",
			url:         "/cameras?id=123",
			expectError: false,
			checkFilter: func(f *analogdb.CameraFilter) bool {
				return f.IDs != nil && len(*f.IDs) == 1 && (*f.IDs)[0] == 123
			},
		},
		{
			name:        "include counts true",
			url:         "/cameras?include_counts=true",
			expectError: false,
			checkFilter: func(f *analogdb.CameraFilter) bool {
				return f.IncludeCounts != nil && *f.IncludeCounts == true
			},
		},
		{
			name:        "exclude zero counts true",
			url:         "/cameras?exclude_zero_counts=true",
			expectError: false,
			checkFilter: func(f *analogdb.CameraFilter) bool {
				return f.ExcludeZeroCounts != nil && *f.ExcludeZeroCounts == true
			},
		},
		{
			name:        "exclude zero counts false",
			url:         "/cameras?exclude_zero_counts=false",
			expectError: false,
			checkFilter: func(f *analogdb.CameraFilter) bool {
				return f.ExcludeZeroCounts != nil && *f.ExcludeZeroCounts == false
			},
		},
		{
			name:        "invalid page size",
			url:         "/cameras?page_size=invalid",
			expectError: true,
		},
		{
			name:        "invalid id",
			url:         "/cameras?id=invalid",
			expectError: true,
		},
		{
			name:        "invalid include counts",
			url:         "/cameras?include_counts=invalid",
			expectError: true,
		},
		{
			name:        "invalid exclude zero counts",
			url:         "/cameras?exclude_zero_counts=invalid",
			expectError: true,
		},
		{
			name:        "min count",
			url:         "/cameras?min_count=5",
			expectError: false,
			checkFilter: func(f *analogdb.CameraFilter) bool {
				return f.MinCount != nil && *f.MinCount == 5
			},
		},
		{
			name:        "no min count or top posts by default",
			url:         "/cameras",
			expectError: false,
			checkFilter: func(f *analogdb.CameraFilter) bool {
				return f.MinCount == nil && f.TopPosts == nil
			},
		},
		{
			name:        "invalid min count",
			url:         "/cameras?min_count=invalid",
			expectError: true,
		},
		{
			name:        "zero min count",
			url:         "/cameras?min_count=0",
			expectError: true,
		},
		{
			name:        "top posts",
			url:         "/cameras?top_posts=6",
			expectError: false,
			checkFilter: func(f *analogdb.CameraFilter) bool {
				return f.TopPosts != nil && *f.TopPosts == 6
			},
		},
		{
			name:        "top posts clamped to max",
			url:         "/cameras?top_posts=50",
			expectError: false,
			checkFilter: func(f *analogdb.CameraFilter) bool {
				return f.TopPosts != nil && *f.TopPosts == 10
			},
		},
		{
			name:        "top posts clamped to min",
			url:         "/cameras?top_posts=0",
			expectError: false,
			checkFilter: func(f *analogdb.CameraFilter) bool {
				return f.TopPosts != nil && *f.TopPosts == 1
			},
		},
		{
			name:        "invalid top posts",
			url:         "/cameras?top_posts=invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)

			filter, err := parseToCameraFilter(req)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.checkFilter != nil && !tt.checkFilter(filter) {
				t.Error("filter validation failed")
			}
		})
	}
}

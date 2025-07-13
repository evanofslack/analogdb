package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evanofslack/analogdb"
)

func TestParseToFilmFilter(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		checkFilter func(*analogdb.FilmFilter) bool
	}{
		{
			name:        "default filter",
			url:         "/films",
			expectError: false,
			checkFilter: func(f *analogdb.FilmFilter) bool {
				return f.Sort != nil && *f.Sort == analogdb.FilmSortAlphabetically
			},
		},
		{
			name:        "sort alphabetically",
			url:         "/films?sort=alphabetically",
			expectError: false,
			checkFilter: func(f *analogdb.FilmFilter) bool {
				return f.Sort != nil && *f.Sort == analogdb.FilmSortAlphabetically
			},
		},
		{
			name:        "sort by counts",
			url:         "/films?sort=counts",
			expectError: false,
			checkFilter: func(f *analogdb.FilmFilter) bool {
				return f.Sort != nil && *f.Sort == analogdb.FilmSortCounts
			},
		},
		{
			name:        "invalid sort",
			url:         "/films?sort=invalid",
			expectError: true,
		},
		{
			name:        "page size",
			url:         "/films?page_size=10",
			expectError: false,
			checkFilter: func(f *analogdb.FilmFilter) bool {
				return f.Limit != nil && *f.Limit == 10
			},
		},
		{
			name:        "make filter",
			url:         "/films?make=Kodak",
			expectError: false,
			checkFilter: func(f *analogdb.FilmFilter) bool {
				return f.Make != nil && *f.Make == "Kodak"
			},
		},
		{
			name:        "type filter",
			url:         "/films?type=color",
			expectError: false,
			checkFilter: func(f *analogdb.FilmFilter) bool {
				return f.Type != nil && *f.Type == "color"
			},
		},
		{
			name:        "speed filter",
			url:         "/films?speed=400",
			expectError: false,
			checkFilter: func(f *analogdb.FilmFilter) bool {
				return f.Speed != nil && *f.Speed == 400
			},
		},
		{
			name:        "colortype filter",
			url:         "/films?colortype=bw",
			expectError: false,
			checkFilter: func(f *analogdb.FilmFilter) bool {
				return f.ColorType != nil && *f.ColorType == "bw"
			},
		},
		{
			name:        "id filter",
			url:         "/films?id=123",
			expectError: false,
			checkFilter: func(f *analogdb.FilmFilter) bool {
				return f.IDs != nil && len(*f.IDs) == 1 && (*f.IDs)[0] == 123
			},
		},
		{
			name:        "include counts true",
			url:         "/films?include_counts=true",
			expectError: false,
			checkFilter: func(f *analogdb.FilmFilter) bool {
				return f.IncludeCounts != nil && *f.IncludeCounts == true
			},
		},
		{
			name:        "exclude zero counts",
			url:         "/films?exclude_zero_counts=false",
			expectError: false,
			checkFilter: func(f *analogdb.FilmFilter) bool {
				return f.IncludeCounts != nil && *f.IncludeCounts == false
			},
		},
		{
			name:        "invalid page size",
			url:         "/films?page_size=invalid",
			expectError: true,
		},
		{
			name:        "invalid speed",
			url:         "/films?speed=invalid",
			expectError: true,
		},
		{
			name:        "invalid id",
			url:         "/films?id=invalid",
			expectError: true,
		},
		{
			name:        "invalid include counts",
			url:         "/films?include_counts=invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)

			filter, err := parseToFilmFilter(req)

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

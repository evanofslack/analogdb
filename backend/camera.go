package analogdb

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// CreateCamera is model for creating a camera in database
type CreateCamera struct {
	Id          int       `json:"id"`
	Make        string    `json:"make"`
	Model       string    `json:"model"`
	Description string    `json:"description"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
}

// CameraModel represents a specific camera model with post count
type Camera struct {
	Id          int       `json:"id"`
	Make        string    `json:"make"`
	Model       string    `json:"model"`
	Description string    `json:"description"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
	PostCount   int       `json:"post_count"`
}

type CameraSort int

const (
	CameraSortUnknown CameraSort = iota
	CameraSortAlphabetical
	CameraSortCounts
)

func (s CameraSort) String() string {
	switch s {
	case CameraSortAlphabetical:
		return "alphabetical"
	case CameraSortCounts:
		return "counts"
	default:
		return "unknown"
	}
}

func CameraSortFromString(s string) CameraSort {
	switch strings.ToLower(s) {
	case "alphabetical":
		return CameraSortAlphabetical
	case "counts":
		return CameraSortCounts
	default:
		return CameraSortUnknown
	}
}

// CameraFilter are options used for querying films
type CameraFilter struct {
	Limit             *int
	Sort              *CameraSort
	IDs               *[]int
	Make              *string
	Model             *string
	IncludeCounts     *bool
	ExcludeZeroCounts *bool
}

func NewCameraFilter(limit *int, sort *CameraSort, ids *[]int, make *string, model *string, speed *int, colortype *string, includeCounts *bool, excludeZeroCounts *bool) *CameraFilter {
	return &CameraFilter{
		Limit:             limit,
		Sort:              sort,
		IDs:               ids,
		Make:              make,
		Model:             model,
		IncludeCounts:     includeCounts,
		ExcludeZeroCounts: excludeZeroCounts,
	}
}

func (filter *CameraFilter) String() string {
	out := []string{}
	if limit := filter.Limit; limit != nil {
		out = append(out, fmt.Sprintf("limit: %d", *limit))
	}
	if sort := filter.Sort; sort != nil {
		out = append(out, fmt.Sprintf("sort: %q", *sort))
	}
	if ids := filter.IDs; ids != nil {
		out = append(out, fmt.Sprintf("ids: %v", *ids))
	}
	if make := filter.Make; make != nil {
		out = append(out, fmt.Sprintf("make: %s", *make))
	}
	if model := filter.Model; model != nil {
		out = append(out, fmt.Sprintf("model: %s", *model))
	}
	if includeCounts := filter.IncludeCounts; includeCounts != nil {
		out = append(out, fmt.Sprintf("include_counts: %t", *includeCounts))
	}
	if excludeZeros := filter.ExcludeZeroCounts; excludeZeros != nil {
		out = append(out, fmt.Sprintf("exclude_zero_counts: %t", *excludeZeros))
	}
	return strings.Join(out, ", ")
}

type CameraService interface {
	FindCameras(ctx context.Context, filter *CameraFilter) ([]*Camera, error)
	CreateCamera(ctx context.Context, film *CreateCamera) (*CreateCamera, error)
}

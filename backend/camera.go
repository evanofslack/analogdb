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
	CameraSortAlphabetically
	CameraSortCounts
)

func (s CameraSort) String() string {
	switch s {
	case CameraSortAlphabetically:
		return "alphabetically"
	case CameraSortCounts:
		return "counts"
	default:
		return "unknown"
	}
}

func CameraSortFromString(s string) CameraSort {
	switch strings.ToLower(s) {
	case "alphabetically":
		return CameraSortAlphabetically
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
	if includeCounts := filter.IncludeCounts; includeCounts != nil {
		out = append(out, fmt.Sprintf("include_counts: %t", *includeCounts))
	}
	if excludeZeros := filter.ExcludeZeroCounts; excludeZeros != nil {
		out = append(out, fmt.Sprintf("exclude_zero_counts: %t", *excludeZeros))
	}
	return strings.Join(out, ", ")
}

func NewCameraFilter(limit *int, sort *CameraSort, ids *[]int, make *string, ty *string, speed *int, colortype *string, includeCounts *bool, excludeZeroCounts *bool) *CameraFilter {
	return &CameraFilter{
		Limit:             limit,
		Sort:              sort,
		IDs:               ids,
		Make:              make,
		IncludeCounts:     includeCounts,
		ExcludeZeroCounts: excludeZeroCounts,
	}
}

type CameraService interface {
	AllCameras(ctx context.Context, filter *CameraFilter) ([]*Camera, error)
	CreateCamera(ctx context.Context, film *CreateCamera) (*CreateCamera, error)
}

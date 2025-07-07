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

// Camera represents a camera with its make, models and total count
type Camera struct {
	Make      string        `json:"make"`
	Models    []CameraModel `json:"camera_models"`
	PostCount int           `json:"post_count"`
}

// CameraModel represents a specific camera model with post count
type CameraModel struct {
	Id          int       `json:"id"`
	Make        string    `json:"make"`
	Model       string    `json:"model"`
	Description string    `json:"description"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
	PostCount   int       `json:"post_count"`
}

// CameraFilter are options used for querying films
type CameraFilter struct {
	IncludeCounts     *bool
	ExcludeZeroCounts *bool
}

func (filter *CameraFilter) String() string {
	out := []string{}
	if filter.IncludeCounts != nil {
		out = append(out, fmt.Sprintf("include_counts: %t", *filter.IncludeCounts))
	}
	if filter.ExcludeZeroCounts != nil {
		out = append(out, fmt.Sprintf("exclude_zero_counts: %t", *filter.ExcludeZeroCounts))
	}
	return strings.Join(out, ", ")
}

func NewCameraFilter(includeCounts *bool, excludeZeroCounts *bool) *CameraFilter {
	return &CameraFilter{
		IncludeCounts:     includeCounts,
		ExcludeZeroCounts: excludeZeroCounts,
	}
}

type CameraService interface {
	AllCameras(ctx context.Context, filter *CameraFilter) ([]*Camera, error)
	CreateCamera(ctx context.Context, film *CreateCamera) (*CreateCamera, error)
}

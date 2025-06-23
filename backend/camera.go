package analogdb

import (
	"context"
	"time"
)

// Camera represents the source info for a camera
type Camera struct {
	Id          int    `json:"id"`
	Make        string    `json:"make"`
	Model       string    `json:"model"`
	Description string    `json:"description"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
}

type CameraService interface {
	Cameras(ctx context.Context) ([]*Camera, error)
}

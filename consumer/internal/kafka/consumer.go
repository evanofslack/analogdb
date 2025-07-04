package kafka

import (
	"context"

	v1 "github.com/evanofslack/analogdb-consumer/internal/gen/proto/analytics/v1"
)

type Consumer interface {
	Read(ctx context.Context) ([]v1.Event, error)
	Commit(ctx context.Context, msgs []v1.Event) error
	Close() error
}

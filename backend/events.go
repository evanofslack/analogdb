package analogdb

import (
	"context"

	v1 "github.com/evanofslack/analogdb/internal/gen/proto/analytics/v1"
)

type EventService interface {
	Write(ctx context.Context, event *v1.Event) error
	Close() error
}

type UiEventService interface {
	Write(ctx context.Context, event *v1.UiEvent) error
	Close() error
}

// SaltService stores the salt for a UTC day. The first caller's salt wins and
// every caller gets back the stored one
type SaltService interface {
	DailySalt(ctx context.Context, day string, salt []byte) ([]byte, error)
}

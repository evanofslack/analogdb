package analogdb

import (
	"context"

	v1 "github.com/evanofslack/analogdb/internal/gen/proto/analytics/v1"
)

type EventService interface {
	Write(ctx context.Context, event *v1.Event) error
}

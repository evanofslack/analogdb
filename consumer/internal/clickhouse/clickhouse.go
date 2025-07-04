package clickhouse

import (
	"context"

	v1 "github.com/evanofslack/analogdb-consumer/internal/gen/proto/analytics/v1"
)

type DB interface {
	Insert(ctx context.Context, events []v1.Event) error
	Health(ctx context.Context) error
	Close() error
}

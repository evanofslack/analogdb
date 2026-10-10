package events

import (
	"context"

	v1 "github.com/evanofslack/analogdb/internal/gen/proto/analytics/v1"
	"github.com/evanofslack/analogdb/logger"
)

type Noop[T any] struct {
	logger *logger.Logger
}

type NoopEventStream = Noop[*v1.Event]

type NoopUiEventStream = Noop[*v1.UiEvent]

func NewNoop(logger *logger.Logger) *NoopEventStream {
	es := &NoopEventStream{logger: logger}
	logger.Info("Initialized noop Kafka event stream")
	return es
}

func NewNoopUi(logger *logger.Logger) *NoopUiEventStream {
	es := &NoopUiEventStream{logger: logger}
	logger.Info("Initialized noop Kafka UI event stream")
	return es
}

func (n *Noop[T]) Write(ctx context.Context, event T) error {
	n.logger.Debug("Noop, skip write event to Kafka")
	return nil
}

func (n *Noop[T]) Close() error {
	return nil
}

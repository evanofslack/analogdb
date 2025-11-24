package events

import (
	"context"

	v1 "github.com/evanofslack/analogdb/internal/gen/proto/analytics/v1"
	"github.com/evanofslack/analogdb/logger"
)

type NoopEventStream struct {
	logger *logger.Logger
}

func NewNoop(logger *logger.Logger) *NoopEventStream {
	es := &NoopEventStream{logger: logger}
	logger.Info("Initialized noop Kafka event stream")
	return es
}

func (n *NoopEventStream) Write(ctx context.Context, event *v1.Event) error {
	n.logger.Debug("Noop, skip write event to Kafka")
	return nil
}

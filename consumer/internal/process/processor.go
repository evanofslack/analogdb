package process

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	v1 "github.com/evanofslack/analogdb-consumer/internal/gen/proto/analytics/v1"
	"github.com/segmentio/kafka-go"
)

type Consumer interface {
	Read(ctx context.Context) ([]*v1.Event, []kafka.Message, error)
	Commit(ctx context.Context, msgs []kafka.Message) error
	Close() error
}

type DB interface {
	Insert(ctx context.Context, events []*v1.Event) error
}

type Processor struct {
	logger   *slog.Logger
	consumer Consumer
	db       DB
}

func New(logger *slog.Logger, consumer Consumer, db DB) *Processor {
	return &Processor{
		consumer: consumer,
		db:       db,
		logger:   logger,
	}
}

func (p *Processor) Start(ctx context.Context) error {
	p.logger.Info("Start processor")
	defer p.logger.Info("Stop processor")

	const maxRetries = 5
	retryDelay := time.Second
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		for attempt := 1; attempt <= maxRetries; attempt++ {
			err := p.processBatch(ctx)
			if err == nil {
				break // success
			}
			if isTimeoutError(err) {
				break // timeout, continue
			}
			if attempt >= maxRetries {
				p.logger.Error("Process max retries exceeded", "error", err)
				break // give up
			}
			delay := retryDelay * time.Duration(attempt)
			p.logger.Warn("Process batch failed, retrying", "attempt", attempt, "error", err, "delay", delay)
			select {
			case <-ctx.Done():
				return err
			case <-time.After(delay):
			}
		}
	}
}

func (p *Processor) processBatch(ctx context.Context) error {
	start := time.Now()
	events, messages, err := p.consumer.Read(ctx)
	if err != nil {
		return fmt.Errorf("read events: %w", err)
	}
	p.logger.Debug("Start process batch", "count", len(events))

	if len(events) == 0 {
		return nil
	}

	if err := p.validateEvents(events); err != nil {
		p.logger.Error("Event validation failed", "error", err)
		return fmt.Errorf("validate events: %w", err)
	}

	if err := p.db.Insert(ctx, events); err != nil {
		return fmt.Errorf("insert events: %w", err)
	}

	if err := p.consumer.Commit(ctx, messages); err != nil {
		return fmt.Errorf("commit messages: %w", err)
	}

	duration := time.Since(start)
	p.logger.Debug("Finish process batch",
		"count", len(events),
		"duration_ms", duration.Milliseconds(),
	)

	return nil
}

func (p *Processor) validateEvents(events []*v1.Event) error {
	for i, event := range events {
		if err := p.validateEvent(i, event); err != nil {
			return err
		}
	}
	return nil
}

func (p *Processor) validateEvent(index int, event *v1.Event) error {
	if event == nil {
		return fmt.Errorf("event %d: nil event", index)
	}
	if event.RequestId == "" {
		return fmt.Errorf("event %d: missing request_id", index)
	}
	if event.StartTime <= 0 || event.EndTime <= 0 {
		return fmt.Errorf("event %d: invalid timestamps start=%d end=%d",
			index, event.StartTime, event.EndTime)
	}
	if event.EndTime < event.StartTime {
		return fmt.Errorf("event %d: end_time must be after start_time", index)
	}
	return nil
}

func (p *Processor) Stop() error {
	p.logger.Info("Stopping processor")
	return p.consumer.Close()
}

func isTimeoutError(err error) bool {
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "context deadline exceeded") ||
		strings.Contains(errStr, "request timed out") ||
		strings.Contains(errStr, "no messages received")
}

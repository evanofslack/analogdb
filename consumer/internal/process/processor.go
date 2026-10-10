package process

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	v1 "github.com/evanofslack/analogdb-consumer/internal/gen/proto/analytics/v1"
	"github.com/evanofslack/analogdb-consumer/internal/metrics"
)

const (
	retryDelay      = time.Second
	maxRetryDelay   = 30 * time.Second
	errorLogAttempt = 10
)

type Consumer[T any] interface {
	Read(ctx context.Context) ([]*T, []kafka.Message, error)
	Commit(ctx context.Context, msgs []kafka.Message) error
	Close() error
}

type DB[T any] interface {
	Insert(ctx context.Context, events []*T) error
}

type Processor[T any] struct {
	logger   *slog.Logger
	metrics  *metrics.Metrics
	name     string
	consumer Consumer[T]
	db       DB[T]
	validate func(*T) error
	backoff  func(attempt int) time.Duration
}

func New[T any](logger *slog.Logger, metrics *metrics.Metrics, name string, consumer Consumer[T], db DB[T], validate func(*T) error) *Processor[T] {
	return &Processor[T]{
		logger:   logger.With("topic", name),
		metrics:  metrics,
		name:     name,
		consumer: consumer,
		db:       db,
		validate: validate,
		backoff:  backoff,
	}
}

func (p *Processor[T]) Start(ctx context.Context) error {
	p.logger.Info("Start processor")
	defer p.logger.Info("Stop processor")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := p.processBatch(ctx); err != nil {
			return err
		}
	}
}

func (p *Processor[T]) processBatch(ctx context.Context) error {
	start := time.Now()
	events, messages, err := p.consumer.Read(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if isTimeoutError(err) {
			return nil
		}
		p.logger.Warn("Read batch failed", "error", err)
		return sleep(ctx, p.backoff(1))
	}
	if len(messages) == 0 {
		return nil
	}
	p.logger.Debug("Start process batch", "count", len(events), "messages", len(messages))

	kept := p.filter(events)
	if len(kept) > 0 {
		err := p.retry(ctx, "insert", func() error { return p.db.Insert(ctx, kept) })
		if err != nil {
			return err
		}
	}

	err = p.retry(ctx, "commit", func() error { return p.consumer.Commit(ctx, messages) })
	if err != nil {
		return err
	}

	p.logger.Debug("Finish process batch",
		"count", len(kept),
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return nil
}

func (p *Processor[T]) filter(events []*T) []*T {
	kept := make([]*T, 0, len(events))
	var dropped int
	var lastErr error
	for _, event := range events {
		if err := p.validate(event); err != nil {
			dropped++
			lastErr = err
			continue
		}
		kept = append(kept, event)
	}
	if dropped > 0 {
		p.metrics.IncrementEventsDropped(dropped, p.name, "invalid")
		p.logger.Warn("Dropped invalid events", "count", dropped, "error", lastErr)
	}
	return kept
}

func (p *Processor[T]) retry(ctx context.Context, op string, fn func() error) error {
	var retrying bool
	defer func() {
		if retrying {
			p.metrics.SetBatchRetrying(p.name, false)
		}
	}()

	for attempt := 1; ; attempt++ {
		err := fn()
		if err == nil {
			if retrying {
				p.logger.Info("Batch retry succeeded", "op", op, "attempts", attempt)
			}
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !retrying {
			retrying = true
			p.metrics.SetBatchRetrying(p.name, true)
		}

		delay := p.backoff(attempt)
		if attempt%errorLogAttempt == 0 {
			p.logger.Error("Batch still failing, retrying", "op", op, "attempt", attempt, "error", err, "delay", delay)
		} else {
			p.logger.Warn("Batch failed, retrying", "op", op, "attempt", attempt, "error", err, "delay", delay)
		}
		if err := sleep(ctx, delay); err != nil {
			return err
		}
	}
}

func (p *Processor[T]) Stop() error {
	p.logger.Info("Stopping processor")
	return p.consumer.Close()
}

func ValidateEvent(event *v1.Event) error {
	if event == nil {
		return fmt.Errorf("nil event")
	}
	if event.RequestId == "" {
		return fmt.Errorf("missing request_id")
	}
	if event.StartTime <= 0 || event.EndTime <= 0 {
		return fmt.Errorf("invalid timestamps start=%d end=%d", event.StartTime, event.EndTime)
	}
	if event.EndTime < event.StartTime {
		return fmt.Errorf("end_time must be after start_time")
	}
	return nil
}

func ValidateUiEvent(event *v1.UiEvent) error {
	if event == nil {
		return fmt.Errorf("nil event")
	}
	if event.EventId == "" {
		return fmt.Errorf("missing event_id")
	}
	if _, err := uuid.Parse(event.EventId); err != nil {
		return fmt.Errorf("invalid event_id: %w", err)
	}
	if event.EventName == "" {
		return fmt.Errorf("missing event_name")
	}
	return nil
}

func backoff(attempt int) time.Duration {
	delay := retryDelay * time.Duration(attempt)
	if delay > maxRetryDelay {
		return maxRetryDelay
	}
	return delay
}

func sleep(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func isTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "request timed out") ||
		strings.Contains(errStr, "no messages received")
}

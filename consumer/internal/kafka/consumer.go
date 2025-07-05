package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"

	v1 "github.com/evanofslack/analogdb-consumer/internal/gen/proto/analytics/v1"
)

type Client struct {
	reader    *kafka.Reader
	batchSize int
	timeout   time.Duration
	logger    *slog.Logger
}

func New(logger *slog.Logger, brokers []string, topic, consumerGroup string, batchSize int, timeout time.Duration) *Client {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:          brokers,
		Topic:            topic,
		GroupID:          consumerGroup,
		MinBytes:         10e3,
		MaxBytes:         10e6,
		CommitInterval:   0,
		StartOffset:      kafka.FirstOffset,
		ReadBatchTimeout: time.Millisecond * 200,
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			logger.Error("error", "msg", fmt.Sprintf(msg, args...))
		}),
		Logger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			logger.Debug("debug", "msg", fmt.Sprintf(msg, args...))
		}),
	})

	return &Client{
		reader:    reader,
		batchSize: batchSize,
		timeout:   timeout,
		logger:    logger,
	}
}

func (c *Client) Read(ctx context.Context) ([]*v1.Event, []kafka.Message, error) {
	var events []*v1.Event
	var messages []kafka.Message

	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	for len(events) < c.batchSize {
		select {
		case <-timeoutCtx.Done():
			if len(events) > 0 {
				c.logger.Debug("batch timeout reached", "count", len(events))
				return events, messages, nil
			}
			return nil, nil, timeoutCtx.Err()
		default:
		}

		msg, err := c.reader.FetchMessage(timeoutCtx)
		if err != nil {
			if len(events) > 0 {
				c.logger.Debug("fetch error with partial batch", "error", err, "count", len(events))
				return events, messages, nil
			}
			return nil, nil, fmt.Errorf("fetch message: %w", err)
		}

		event, err := c.deserializeEvent(msg.Value)
		if err != nil {
			c.logger.Error("failed to deserialize event", "error", err, "offset", msg.Offset)
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				c.logger.Error("failed to commit bad message", "error", err)
			}
			continue
		}

		events = append(events, event)
		messages = append(messages, msg)
	}

	c.logger.Debug("batch read complete", "count", len(events))
	return events, messages, nil
}

func (c *Client) Commit(ctx context.Context, msgs []kafka.Message) error {
	if len(msgs) == 0 {
		return nil
	}

	if err := c.reader.CommitMessages(ctx, msgs...); err != nil {
		return fmt.Errorf("commit messages: %w", err)
	}

	c.logger.Debug("committed messages", "count", len(msgs))
	return nil
}

func (c *Client) Close() error {
	return c.reader.Close()
}

func (c *Client) HealthCheck(ctx context.Context) error {
	stats := c.reader.Stats()
	if stats.Partition == "" {
		return fmt.Errorf("kafka consumer not initialized")
	}
	return nil
}

func (c *Client) deserializeEvent(data []byte) (*v1.Event, error) {
	event := &v1.Event{}
	if err := proto.Unmarshal(data, event); err != nil {
		return nil, fmt.Errorf("unmarshal proto: %w", err)
	}
	return event, nil
}

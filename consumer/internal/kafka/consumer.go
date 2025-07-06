package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	v1 "github.com/evanofslack/analogdb-consumer/internal/gen/proto/analytics/v1"
)

type Client struct {
	logger    *slog.Logger
	reader    *kafka.Reader
	batchSize int
	timeout   time.Duration
}

func New(logger *slog.Logger, brokers []string, topic, consumerGroup string, batchSize int, timeout time.Duration) *Client {
	logger = logger.With("brokers", brokers, "consumer_group", consumerGroup, "batch_size", batchSize, "timeout", timeout)
	logger.Debug("Start create new kafka client")
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

	logger.Info("Finish create new kafka client")
	return &Client{
		logger:    logger,
		reader:    reader,
		batchSize: batchSize,
		timeout:   timeout,
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
				c.logger.Debug("Batch timeout reached", "count", len(events))
				return events, messages, nil
			}
			return nil, nil, timeoutCtx.Err()
		default:
		}

		msg, err := c.reader.FetchMessage(timeoutCtx)
		if err != nil {
			if len(events) > 0 {
				c.logger.Debug("Fetch error with partial batch", "error", err, "count", len(events))
				return events, messages, nil
			}
			return nil, nil, fmt.Errorf("fetch message: %w", err)
		}

		event, err := c.deserializeEvent(msg.Value)
		if err != nil {
			c.logger.Error("Deserialize event", "error", err, "offset", msg.Offset)
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				c.logger.Error("Commit bad message", "error", err)
			}
			continue
		}

		events = append(events, event)
		messages = append(messages, msg)
	}

	c.logger.Debug("Batch read complete", "count", len(events))
	return events, messages, nil
}

func (c *Client) Commit(ctx context.Context, msgs []kafka.Message) error {
	if len(msgs) == 0 {
		return nil
	}

	if err := c.reader.CommitMessages(ctx, msgs...); err != nil {
		return fmt.Errorf("commit messages: %w", err)
	}

	c.logger.Debug("Committed messages", "count", len(msgs))
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
	if err := json.Unmarshal(data, event); err != nil {
		return nil, fmt.Errorf("unmarshal json: %w", err)
	}
	return event, nil
}

package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/evanofslack/analogdb-consumer/internal/metrics"
)

type Client[T any] struct {
	logger        *slog.Logger
	metrics       *metrics.Metrics
	reader        *kafka.Reader
	brokers       []string
	consumerGroup string
	topic         string
	batchSize     int
	timeout       time.Duration
}

func New[T any](logger *slog.Logger, metrics *metrics.Metrics, brokers []string, topic, consumerGroup string, batchSize int, timeout time.Duration) *Client[T] {
	logger = logger.With("brokers", brokers, "topic", topic, "consumer_group", consumerGroup, "batch_size", batchSize, "timeout", timeout)
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
	return &Client[T]{
		logger:        logger,
		metrics:       metrics,
		reader:        reader,
		brokers:       brokers,
		consumerGroup: consumerGroup,
		topic:         topic,
		batchSize:     batchSize,
		timeout:       timeout,
	}
}

func (c *Client[T]) Read(ctx context.Context) ([]*T, []kafka.Message, error) {
	var events []*T
	var messages []kafka.Message

	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	for len(messages) < c.batchSize {
		select {
		case <-timeoutCtx.Done():
			if len(messages) > 0 {
				c.logger.Debug("Batch timeout reached", "count", len(events))
				return events, messages, nil
			}
			return nil, nil, timeoutCtx.Err()
		default:
		}

		msg, err := c.reader.FetchMessage(timeoutCtx)
		if err != nil {
			c.metrics.IncrementEventsRead(1, c.consumerGroup, c.topic, err)
			if len(messages) > 0 {
				c.logger.Debug("Fetch error with partial batch", "error", err, "count", len(events))
				return events, messages, nil
			}
			return nil, nil, fmt.Errorf("fetch message: %w", err)
		}
		c.metrics.IncrementEventsRead(1, c.consumerGroup, c.topic, nil)

		messages = append(messages, msg)
		event, err := c.deserializeEvent(msg.Value)
		if err != nil {
			c.logger.Error("Deserialize event", "error", err, "offset", msg.Offset)
			c.metrics.IncrementEventsDropped(1, c.topic, "decode")
			continue
		}
		events = append(events, event)
	}

	c.logger.Debug("Batch read complete", "count", len(events))
	return events, messages, nil
}

func (c *Client[T]) Commit(ctx context.Context, msgs []kafka.Message) error {
	if len(msgs) == 0 {
		return nil
	}

	if err := c.reader.CommitMessages(ctx, msgs...); err != nil {
		c.metrics.IncrementEventsCommitted(len(msgs), c.consumerGroup, c.topic, err)
		return fmt.Errorf("commit messages: %w", err)
	}

	c.metrics.IncrementEventsCommitted(len(msgs), c.consumerGroup, c.topic, nil)
	c.logger.Debug("Committed messages", "count", len(msgs))
	return nil
}

func (c *Client[T]) Close() error {
	return c.reader.Close()
}

func (c *Client[T]) HealthCheck(ctx context.Context) error {
	stats := c.reader.Stats()
	if stats.Partition == "" {
		return fmt.Errorf("kafka consumer not initialized")
	}
	return nil
}

func (c *Client[T]) deserializeEvent(data []byte) (*T, error) {
	event := new(T)
	if err := json.Unmarshal(data, event); err != nil {
		return nil, fmt.Errorf("unmarshal json: %w", err)
	}
	return event, nil
}

func (c *Client[T]) EnsureTopic(ctx context.Context) error {
	if len(c.brokers) == 0 {
		return fmt.Errorf("no kafka brokers")
	}
	var dialer kafka.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", c.brokers[0])
	if err != nil {
		return fmt.Errorf("dial kafka: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	partitions, err := conn.ReadPartitions(c.topic)
	if err == nil && len(partitions) > 0 {
		c.logger.Debug("Kafka topic already exists", "partitions", len(partitions))
		return nil
	}

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("get controller: %w", err)
	}
	controllerConn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return fmt.Errorf("connect to controller broker: %w", err)
	}
	defer func() { _ = controllerConn.Close() }()
	if deadline, ok := ctx.Deadline(); ok {
		_ = controllerConn.SetDeadline(deadline)
	}

	err = controllerConn.CreateTopics(kafka.TopicConfig{
		Topic:             c.topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
		ConfigEntries: []kafka.ConfigEntry{
			{ConfigName: "retention.ms", ConfigValue: "604800000"},
			{ConfigName: "cleanup.policy", ConfigValue: "delete"},
		},
	})
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			c.logger.Debug("Kafka topic already exists")
			return nil
		}
		return fmt.Errorf("create topic: %w", err)
	}

	c.logger.Info("Created kafka topic")
	return nil
}

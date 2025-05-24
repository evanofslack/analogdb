package events

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	v1 "github.com/evanofslack/analogdb/internal/gen/proto/analytics/v1"
	"github.com/evanofslack/analogdb/logger"
	"github.com/segmentio/kafka-go"
)

const (
	batchSize    = 1
	batchTimeout = 1 * time.Second
	async        = false
)

// EventStream is a wrapper around kafka-go to produce events
type EventStream struct {
	logger    *logger.Logger
	writer    *kafka.Writer
	topic     string
	brokers   []string
	connected bool
}

func New(logger *logger.Logger, topic string, brokers []string) (*EventStream, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("no kafka brokers provided")
	}

	if topic == "" {
		return nil, fmt.Errorf("no kafka topic provided")
	}

	// Ensure the topic exists before creating the writer
	if err := createTopicIfNotExist(logger, topic, brokers); err != nil {
		logger.Warn().Err(err).Str("topic", topic).Msg("Failed to create topic")
	}

	addr := kafka.TCP(brokers...)
	writer := &kafka.Writer{
		Addr:         addr,
		Topic:        topic,
		BatchSize:    batchSize,
		BatchTimeout: batchTimeout,
		Async:        async,
	}

	es := &EventStream{
		logger:    logger,
		writer:    writer,
		topic:     topic,
		brokers:   brokers,
		connected: true,
	}

	logger.Info().Str("topic", topic).Strs("brokers", brokers).Str("addr", addr.String()).Int("batch_size", batchSize).Bool("async", async).Msg("Initialized kafka event stream")
	return es, nil
}

// Write serializes and writes record to kafka
func (es *EventStream) Write(ctx context.Context, e *v1.Event) error {
	es.logger.Debug().Str("topic", es.topic).Strs("brokers", es.brokers).Msg("Start write event to kafka")
	if !es.connected {
		return fmt.Errorf("event stream not connected")
	}

	value, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal record, err=%w", err)
	}
	msg := kafka.Message{
		Value: value,
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := es.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write to kafka, topic=%s, err=%w", es.writer.Topic, err)
	}
	es.logger.Debug().Str("topic", es.topic).Strs("brokers", es.brokers).Msg("Finish write event to kafka")
	return nil
}

// Close closes the kafka writer
func (es *EventStream) Close() error {
	if !es.connected {
		return nil
	}
	es.connected = false
	es.logger.Info().Str("topic", es.topic).Msg("Closing kafka event stream")
	return es.writer.Close()
}

func createTopicIfNotExist(logger *logger.Logger, topic string, brokers []string) error {
	conn, err := kafka.Dial("tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("failed to dial kafka: %w", err)
	}
	defer conn.Close()

	// Check if topic exists
	partitions, err := conn.ReadPartitions(topic)
	if err == nil && len(partitions) > 0 {
		logger.Debug().Str("topic", topic).Int("partitions", len(partitions)).Msg("Kafka topic already exists")
		return nil
	}

	// connect to leader
	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("failed to get controller: %w", err)
	}
	controllerConn, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return fmt.Errorf("failed to connect to controller broker: %w", err)
	}
	defer controllerConn.Close()

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
			ConfigEntries: []kafka.ConfigEntry{
				{
					ConfigName:  "retention.ms",
					ConfigValue: "604800000", // 7 days
				},
				{
					ConfigName:  "cleanup.policy",
					ConfigValue: "delete",
				},
			},
		},
	}

	// Create topic
	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			logger.Debug().Str("topic", topic).Msg("Topic already exists")
			return nil
		}
		return fmt.Errorf("failed to create topic, err=%w", err)
	}

	logger.Info().Str("topic", topic).Msg("Successfully created kafka topic")
	return nil
}

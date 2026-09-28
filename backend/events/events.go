package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	v1 "github.com/evanofslack/analogdb/internal/gen/proto/analytics/v1"
	"github.com/evanofslack/analogdb/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/kafka-go"
)

const (
	defaultQueueSize    = 10_000
	defaultBatchSize    = 100
	defaultBatchTimeout = 1 * time.Second
	writerBatchTimeout  = 10 * time.Millisecond
	writeTimeout        = 10 * time.Second
	drainTimeout        = 5 * time.Second
	errorLogInterval    = 30 * time.Second
)

var ErrQueueFull = errors.New("event queue full")
var ErrClosed = errors.New("event stream closed")

// MessageWriter is the subset of kafka.Writer used by EventStream
type MessageWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

type Options struct {
	QueueSize    int
	BatchSize    int
	BatchTimeout time.Duration
}

func (o *Options) setDefaults() {
	if o.QueueSize <= 0 {
		o.QueueSize = defaultQueueSize
	}
	if o.BatchSize <= 0 {
		o.BatchSize = defaultBatchSize
	}
	if o.BatchTimeout <= 0 {
		o.BatchTimeout = defaultBatchTimeout
	}
}

// EventStream queues events in memory and writes them to kafka in the background
type EventStream struct {
	logger  *logger.Logger
	writer  MessageWriter
	topic   string
	brokers []string
	opts    Options
	stats   *eventStats

	queue  chan *v1.Event
	done   chan struct{}
	ctx    context.Context
	cancel context.CancelFunc

	mu     sync.RWMutex
	closed bool

	lastErrorLog time.Time
}

func New(logger *logger.Logger, registerer prometheus.Registerer, topic string, brokers []string, opts Options) (*EventStream, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("no kafka brokers provided")
	}

	if topic == "" {
		return nil, fmt.Errorf("no kafka topic provided")
	}
	opts.setDefaults()

	// Ensure the topic exists before creating the writer
	if err := createTopicIfNotExist(logger, topic, brokers); err != nil {
		logger.Warn("Fail create topic", "error", err, "topic", topic)
	}

	addr := kafka.TCP(brokers...)
	writer := &kafka.Writer{
		Addr:         addr,
		Topic:        topic,
		BatchSize:    opts.BatchSize,
		BatchTimeout: writerBatchTimeout,
		Async:        false,
	}

	es, err := NewWithWriter(logger, registerer, writer, opts)
	if err != nil {
		return nil, err
	}
	es.topic = topic
	es.brokers = brokers

	logger.Info("Initialized kafka event stream", "brokers", brokers, "addr", addr.String(), "queue_size", opts.QueueSize, "batch_size", opts.BatchSize, "batch_timeout", opts.BatchTimeout)
	return es, nil
}

// NewWithWriter creates an event stream on top of any message writer and starts the background worker
func NewWithWriter(logger *logger.Logger, registerer prometheus.Registerer, writer MessageWriter, opts Options) (*EventStream, error) {
	opts.setDefaults()

	es := &EventStream{
		logger: logger,
		writer: writer,
		opts:   opts,
		queue:  make(chan *v1.Event, opts.QueueSize),
		done:   make(chan struct{}),
	}
	es.ctx, es.cancel = context.WithCancel(context.Background())

	es.stats = newEventStats(func() float64 { return float64(len(es.queue)) })
	if err := es.stats.register(registerer); err != nil {
		es.cancel()
		return nil, fmt.Errorf("register event metrics: %w", err)
	}

	go es.run()
	return es, nil
}

// Write enqueues the event without blocking. If the queue is full the event is dropped.
func (es *EventStream) Write(_ context.Context, e *v1.Event) error {
	es.mu.RLock()
	defer es.mu.RUnlock()

	if es.closed {
		es.stats.dropped.Inc()
		return ErrClosed
	}

	select {
	case es.queue <- e:
		es.stats.enqueued.Inc()
		return nil
	default:
		es.stats.dropped.Inc()
		return ErrQueueFull
	}
}

func (es *EventStream) run() {
	defer close(es.done)

	ticker := time.NewTicker(es.opts.BatchTimeout)
	defer ticker.Stop()

	batch := make([]kafka.Message, 0, es.opts.BatchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		es.writeBatch(batch)
		batch = batch[:0]
	}

	for {
		select {
		case e, ok := <-es.queue:
			if !ok {
				flush()
				return
			}
			value, err := json.Marshal(e)
			if err != nil {
				es.stats.writeErrors.Inc()
				es.logger.Error("Fail marshal event", "error", err)
				continue
			}
			batch = append(batch, kafka.Message{Value: value})
			if len(batch) >= es.opts.BatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (es *EventStream) writeBatch(batch []kafka.Message) {
	if es.ctx.Err() != nil {
		es.stats.writeErrors.Add(float64(len(batch)))
		return
	}

	ctx, cancel := context.WithTimeout(es.ctx, writeTimeout)
	defer cancel()

	start := time.Now()
	err := es.writer.WriteMessages(ctx, batch...)
	es.stats.writeDuration.Observe(time.Since(start).Seconds())

	if err != nil {
		es.stats.writeErrors.Add(float64(len(batch)))
		if time.Since(es.lastErrorLog) >= errorLogInterval {
			es.lastErrorLog = time.Now()
			es.logger.Error("Fail write events to kafka", "error", err, "topic", es.topic, "count", len(batch))
		} else {
			es.logger.Debug("Fail write events to kafka", "error", err, "topic", es.topic, "count", len(batch))
		}
		return
	}
	es.logger.Debug("Wrote events to kafka", "topic", es.topic, "count", len(batch))
}

// Close stops accepting events, drains the queue and closes the kafka writer
func (es *EventStream) Close() error {
	es.mu.Lock()
	if es.closed {
		es.mu.Unlock()
		return nil
	}
	es.closed = true
	close(es.queue)
	es.mu.Unlock()

	es.logger.Info("Draining kafka event stream", "topic", es.topic, "queued", len(es.queue))

	timer := time.NewTimer(drainTimeout)
	defer timer.Stop()

	select {
	case <-es.done:
		es.logger.Info("Drained kafka event stream", "topic", es.topic)
	case <-timer.C:
		es.cancel()
		<-es.done
		es.logger.Warn("Timed out draining kafka event stream", "topic", es.topic)
	}
	es.cancel()

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
		logger.Debug("Kafka topic already exists", "topic", topic, "partitions", len(partitions))
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
			logger.Debug("Topic already exists", "topic", topic)
			return nil
		}
		return fmt.Errorf("failed to create topic, err=%w", err)
	}

	logger.Info("Successfully created kafka topic", "topic", topic)
	return nil
}

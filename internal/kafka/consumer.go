package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// Consumer defines the interface for Kafka consumption
type Consumer interface {
	Start(ctx context.Context) error
	Close() error
	IsHealthy() bool
	Stats() ConsumerStats
}

// ConsumerStats holds statistics about the consumer
type ConsumerStats struct {
	MessagesConsumed int64
	MessagesErrors   int64
	LastMessageTime  time.Time
	Connected        bool
}

// MessageHandler is a function that processes Kafka messages
type MessageHandler func(topic string, key []byte, value []byte) error

// KafkaReaderConsumer implements the Consumer interface using segmentio/kafka-go
type KafkaReaderConsumer struct {
	brokers []string
	groupID string
	topics  []string
	handler MessageHandler
	reader  *kafka.Reader
	logger  *slog.Logger

	stats   ConsumerStats
	statsMu sync.RWMutex
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// ConsumerConfig holds configuration for the Kafka consumer
type ConsumerConfig struct {
	Brokers           []string
	GroupID           string
	Topics            []string
	Handler           MessageHandler
	InitialOffset     string
	SessionTimeout    time.Duration
	HeartbeatInterval time.Duration
	MaxProcessingTime time.Duration
	RebalanceTimeout  time.Duration
	FetchMin          int32
	FetchMax          int32
	FetchDefault      int32
}

// DefaultConsumerConfig returns a consumer config with sensible defaults
func DefaultConsumerConfig() *ConsumerConfig {
	return &ConsumerConfig{
		InitialOffset:     "latest",
		SessionTimeout:    20 * time.Second,
		HeartbeatInterval: 6 * time.Second,
		MaxProcessingTime: 5 * time.Minute,
		RebalanceTimeout:  60 * time.Second,
		FetchMin:          1,
		FetchMax:          1024 * 1024, // 1MB
		FetchDefault:      1024 * 1024, // 1MB
	}
}

// NewKafkaReaderConsumer creates a new Kafka consumer using kafka-go
func NewKafkaReaderConsumer(config *ConsumerConfig, logger *slog.Logger) (*KafkaReaderConsumer, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if len(config.Brokers) == 0 {
		return nil, fmt.Errorf("brokers cannot be empty")
	}

	if config.GroupID == "" {
		return nil, fmt.Errorf("group_id cannot be empty")
	}

	if len(config.Topics) == 0 {
		return nil, fmt.Errorf("topics cannot be empty")
	}

	if config.Handler == nil {
		return nil, fmt.Errorf("handler cannot be nil")
	}

	if config.InitialOffset == "" {
		config.InitialOffset = "latest"
	}

	startOffset := getInitialOffset(config.InitialOffset)

	consumer := &KafkaReaderConsumer{
		brokers: config.Brokers,
		groupID: config.GroupID,
		topics:  config.Topics,
		handler: config.Handler,
		logger:  logger,
		stats: ConsumerStats{
			Connected: false,
		},
	}

	// Create kafka.Reader configuration
	readerConfig := kafka.ReaderConfig{
		Brokers:           config.Brokers,
		GroupID:           config.GroupID,
		GroupTopics:       config.Topics,
		StartOffset:       startOffset,
		SessionTimeout:    config.SessionTimeout,
		HeartbeatInterval: config.HeartbeatInterval,
		MaxWait:           config.MaxProcessingTime,
		RebalanceTimeout:  config.RebalanceTimeout,
		MinBytes:          int(config.FetchMin),
		MaxBytes:          int(config.FetchMax),
		ReadBackoffMin:    100 * time.Millisecond,
		ReadBackoffMax:    5 * time.Second,
		// Auto-commit enabled
		CommitInterval: time.Second,
	}

	consumer.reader = kafka.NewReader(readerConfig)

	return consumer, nil
}

func (c *KafkaReaderConsumer) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	c.setConnected(true)
	c.logger.Info("kafka consumer started",
		"brokers", c.brokers,
		"group_id", c.groupID,
		"topics", c.topics)

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-ctx.Done():
				c.logger.Info("kafka consumer context cancelled, stopping")
				return
			default:
				msg, err := c.reader.FetchMessage(ctx)
				if err != nil {
					if ctx.Err() != nil {
						// Context was cancelled, exit
						return
					}

					c.logger.Error("error fetching message", "error", err)
					c.incrementMessagesErrors()
					continue
				}

				if err := c.handler(msg.Topic, msg.Key, msg.Value); err != nil {
					c.logger.Error("error processing message",
						"topic", msg.Topic,
						"partition", msg.Partition,
						"offset", msg.Offset,
						"error", err)
					c.incrementMessagesErrors()
				} else {
					c.incrementMessagesConsumed()
				}

				if err := c.reader.CommitMessages(ctx, msg); err != nil {
					c.logger.Error("error committing message",
						"topic", msg.Topic,
						"partition", msg.Partition,
						"offset", msg.Offset,
						"error", err)
				}
			}
		}
	}()

	return nil
}

// Close gracefully shuts down the consumer
func (c *KafkaReaderConsumer) Close() error {
	c.logger.Info("closing kafka consumer")

	if c.cancel != nil {
		c.cancel()
	}

	c.wg.Wait()

	if c.reader != nil {
		if err := c.reader.Close(); err != nil {
			c.logger.Error("error closing reader", "error", err)
			return err
		}
	}

	c.setConnected(false)
	c.logger.Info("kafka consumer closed")
	return nil
}

// IsHealthy returns true if the consumer is connected and consuming
func (c *KafkaReaderConsumer) IsHealthy() bool {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()
	return c.stats.Connected
}

// Stats returns current consumer statistics
func (c *KafkaReaderConsumer) Stats() ConsumerStats {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()
	return c.stats
}

// incrementMessagesConsumed increments the consumed message counter
func (c *KafkaReaderConsumer) incrementMessagesConsumed() {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	c.stats.MessagesConsumed++
	c.stats.LastMessageTime = time.Now()
}

// incrementMessagesErrors increments the error counter
func (c *KafkaReaderConsumer) incrementMessagesErrors() {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	c.stats.MessagesErrors++
}

// setConnected sets the connected status
func (c *KafkaReaderConsumer) setConnected(connected bool) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	c.stats.Connected = connected
}

// getInitialOffset converts string offset to kafka-go offset
func getInitialOffset(offset string) int64 {
	switch offset {
	case "earliest":
		return kafka.FirstOffset
	case "latest":
		return kafka.LastOffset
	default:
		return kafka.LastOffset
	}
}

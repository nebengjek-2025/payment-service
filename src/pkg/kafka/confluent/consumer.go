package kafka

import (
	"context"
	"fmt"
	"notification-service/src/pkg/log"
	"strings"
	"sync"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type consumer struct {
	sync.Mutex
	handler  ConsumerHandler
	consumer *kafka.Consumer
	logger   log.Log
}

// NewConsumer is a constructor of kafka consumer
func NewConsumer(config *kafka.ConfigMap, log log.Log) (Consumer, error) {
	c, err := kafka.NewConsumer(config)
	if err != nil {
		return nil, err
	}

	return &consumer{
		logger:   log,
		consumer: c,
	}, nil
}

func (c *consumer) SetHandler(handler ConsumerHandler) {
	c.handler = handler
}

func (c *consumer) Subscribe(ctx context.Context, topics ...string) {
	if c.handler == nil {
		joinTopic := strings.Join(topics, ", ")
		msg := fmt.Sprintf("Kafka Consumer Error: Topics: [%s] There is no consumer handler to handle message from incoming event", joinTopic)
		c.logger.Error("kafka-consumer", msg, "Subscribe", "")
		return
	}

	if err := c.consumer.SubscribeTopics(topics, nil); err != nil {
		c.logger.Error("kafka-consumer", fmt.Sprintf("Failed to subscribe topics %v: %v", topics, err), "Subscribe", "")
		return
	}

	go func() {
		c.logger.Info("kafka-consumer", fmt.Sprintf("Start consuming topics: %v", topics), "Subscribe", "")

		for {
			select {
			case <-ctx.Done():
				c.logger.Info("kafka-consumer", "Stopping consumer because context is done", "Subscribe", "")
				return
			default:
				msg, err := c.consumer.ReadMessage(-1)
				if err != nil {
					c.logger.Error("kafka-consumer", fmt.Sprintf("Kafka Consumer Error: %v", err), "ReadMessage", "")
					continue
				}

				// Kalau handler tidak thread-safe & kamu mau banyak goroutine, baru butuh lock.
				c.handler.HandleMessage(msg)

				if _, err := c.consumer.CommitMessage(msg); err != nil {
					c.logger.Error("kafka-consumer", fmt.Sprintf("Failed to commit message: %v", err), "CommitMessage", "")
				}
			}
		}
	}()
}

func (c *consumer) Close() error {
	return c.consumer.Close()
}

package messaging

import (
	"fmt"
	kafkaPkg "payment-service/src/pkg/kafka/confluent"
	"payment-service/src/pkg/log"

	k "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type TopicRouterHandler struct {
	logger         log.Log
	handlers       map[string]kafkaPkg.ConsumerHandler
	defaultHandler kafkaPkg.ConsumerHandler
}

func NewTopicRouterHandler(logger log.Log) *TopicRouterHandler {
	return &TopicRouterHandler{
		logger:   logger,
		handlers: make(map[string]kafkaPkg.ConsumerHandler),
	}
}

func (r *TopicRouterHandler) Register(topic string, handler kafkaPkg.ConsumerHandler) {
	r.handlers[topic] = handler
}

func (r *TopicRouterHandler) SetDefault(handler kafkaPkg.ConsumerHandler) {
	r.defaultHandler = handler
}

func (r *TopicRouterHandler) HandleMessage(msg *k.Message) {
	if msg.TopicPartition.Topic == nil {
		r.logger.Error("topic-router",
			"Received message with nil topic",
			"HandleMessage",
			"",
		)
		return
	}

	topic := *msg.TopicPartition.Topic

	if handler, ok := r.handlers[topic]; ok {
		handler.HandleMessage(msg)
		return
	}

	if r.defaultHandler != nil {
		r.logger.Info("topic-router",
			fmt.Sprintf("Using default handler for topic: %s", topic),
			"HandleMessage",
			"",
		)
		r.defaultHandler.HandleMessage(msg)
		return
	}

	r.logger.Error("topic-router",
		fmt.Sprintf("No handler registered for topic: %s", topic),
		"HandleMessage",
		"",
	)
}

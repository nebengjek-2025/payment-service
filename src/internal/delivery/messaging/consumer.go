package messaging

import (
	"context"
	kafkaPkgConfluent "notification-service/src/pkg/kafka/confluent"
	"notification-service/src/pkg/log"
)

type RouterConsumerConfig struct {
	Ctx      context.Context
	Consumer kafkaPkgConfluent.Consumer
	Logger   log.Log
	Handlers map[string]kafkaPkgConfluent.ConsumerHandler
}

func (r RouterConsumerConfig) Setup() {
	router := NewTopicRouterHandler(r.Logger)

	for topic, handler := range r.Handlers {
		router.Register(topic, handler)
	}

	topics := make([]string, 0, len(r.Handlers))
	for topic := range r.Handlers {
		topics = append(topics, topic)
	}

	r.Consumer.SetHandler(router)
	r.Consumer.Subscribe(r.Ctx, topics...)
}

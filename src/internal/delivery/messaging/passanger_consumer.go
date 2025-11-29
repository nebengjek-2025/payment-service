package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"notification-service/src/internal/model"
	"notification-service/src/internal/usecase"
	"notification-service/src/pkg/log"
	"time"

	k "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type PassangerConsumerHandler struct {
	logger           log.Log
	PassangerUseCase *usecase.PassengerUseCase
}

func NewPassangerConsumerHandler(
	logger log.Log,
	passangerUsecase *usecase.PassengerUseCase,
) *PassangerConsumerHandler {
	return &PassangerConsumerHandler{
		logger:           logger,
		PassangerUseCase: passangerUsecase,
	}
}

func (h *PassangerConsumerHandler) HandleMessage(message *k.Message) {
	h.logger.Info(
		"passanger-consumer",
		fmt.Sprintf("Received message: %s", string(message.Value)),
		"HandleMessage",
		"",
	)

	var event model.NotificationUser
	if err := json.Unmarshal(message.Value, &event); err != nil {
		h.logger.Error(
			"passanger-consumer",
			fmt.Sprintf("Failed to unmarshal message: %v", err),
			"HandleMessage",
			"",
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := h.PassangerUseCase.SendNotificationPassanger(ctx, &event)
	if err != nil {
		h.logger.Error(
			"passanger-consumer",
			fmt.Sprintf("Failed to handle passanger event: %v", err),
			"HandleMessage",
			"",
		)
		return
	}

	h.logger.Info(
		"passanger-consumer",
		fmt.Sprintf("Successfully processed passanger event: %+v", event),
		"HandleMessage",
		"",
	)
}

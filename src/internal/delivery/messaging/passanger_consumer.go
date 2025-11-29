package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"notification-service/src/internal/usecase"
	"notification-service/src/pkg/log"
	"time"

	k "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type PassengerConsumerHandler struct {
	logger           log.Log
	passengerUseCase *usecase.PassengerUseCase
}

func NewPassengerConsumerHandler(
	logger log.Log,
	passengerUseCase *usecase.PassengerUseCase,
) *PassengerConsumerHandler {
	return &PassengerConsumerHandler{
		logger:           logger,
		passengerUseCase: passengerUseCase,
	}
}

type PassengerEvent struct {
	PassengerID string `json:"passenger_id"`
	Status      string `json:"status"`
}

func (h *PassengerConsumerHandler) HandleMessage(message *k.Message) {
	h.logger.Info(
		"passenger-consumer",
		fmt.Sprintf("Received message: %s", string(message.Value)),
		"HandleMessage",
		"",
	)

	var evt PassengerEvent
	if err := json.Unmarshal(message.Value, &evt); err != nil {
		h.logger.Error(
			"passenger-consumer",
			fmt.Sprintf("Failed to unmarshal message: %v", err),
			"HandleMessage",
			"",
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := h.passengerUseCase.HandlePassengerEvent(ctx, usecase.HandlePassengerEventRequest{
		PassengerID: evt.PassengerID,
		Status:      evt.Status,
		RawMessage:  string(message.Value),
	})
	if err != nil {
		h.logger.Error(
			"passenger-consumer",
			fmt.Sprintf("Failed to handle passenger event: %v", err),
			"HandleMessage",
			"",
		)
		return
	}

	h.logger.Info(
		"passenger-consumer",
		fmt.Sprintf("Successfully processed passenger event: %+v", evt),
		"HandleMessage",
		"",
	)
}

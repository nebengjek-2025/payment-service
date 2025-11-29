package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"payment-service/src/internal/model"
	"payment-service/src/internal/usecase"
	"payment-service/src/pkg/log"
	"time"

	k "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type DriverConsumerHandler struct {
	logger        log.Log
	DriverUseCase *usecase.DriverUseCase
}

func NewDriverConsumerHandler(
	logger log.Log,
	driverUsecase *usecase.DriverUseCase,
) *DriverConsumerHandler {
	return &DriverConsumerHandler{
		logger:        logger,
		DriverUseCase: driverUsecase,
	}
}

func (h *DriverConsumerHandler) HandleMessage(message *k.Message) {
	h.logger.Info(
		"driver-consumer",
		fmt.Sprintf("Received message: %s", string(message.Value)),
		"HandleMessage",
		"",
	)

	var msg model.DriverEvent
	if err := json.Unmarshal(message.Value, &msg); err != nil {
		h.logger.Error(
			"driver-consumer",
			fmt.Sprintf("Failed to unmarshal message: %v", err),
			"HandleMessage",
			"",
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := h.DriverUseCase.SendNotificationDriver(ctx, msg)
	if err != nil {
		h.logger.Error(
			"driver-consumer",
			fmt.Sprintf("Failed to handle driver event: %v", err),
			"HandleMessage",
			"",
		)
		return
	}

	h.logger.Info(
		"driver-consumer",
		fmt.Sprintf("Successfully processed driver event: %+v", msg),
		"HandleMessage",
		"",
	)
}

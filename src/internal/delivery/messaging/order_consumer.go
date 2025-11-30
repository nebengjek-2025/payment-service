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

type OrderConsumerHandler struct {
	logger       log.Log
	OrderUseCase *usecase.WalletUseCase
}

func NewOrderConsumerHandler(
	logger log.Log,
	walletUsecase *usecase.WalletUseCase,
) *OrderConsumerHandler {
	return &OrderConsumerHandler{
		logger:       logger,
		OrderUseCase: walletUsecase,
	}
}

func (h *OrderConsumerHandler) HandleMessage(message *k.Message) {
	h.logger.Info(
		"order-consumer",
		fmt.Sprintf("Received message: %s", string(message.Value)),
		"HandleMessage",
		"",
	)

	var event model.OrderNotificationEvent
	if err := json.Unmarshal(message.Value, &event); err != nil {
		h.logger.Error(
			"order-consumer",
			fmt.Sprintf("Failed to unmarshal message: %v", err),
			"HandleMessage",
			"",
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := h.OrderUseCase.HoldWalletForOrder(ctx, &event)
	if err != nil {
		h.logger.Error(
			"order-consumer",
			fmt.Sprintf("Failed to handle order event: %v", err),
			"HandleMessage",
			"",
		)
		return
	}

	h.logger.Info(
		"order-consumer",
		fmt.Sprintf("Successfully processed order event: %+v", event),
		"HandleMessage",
		"",
	)
}

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

type WalletConsumerHandler struct {
	logger        log.Log
	WalletUseCase *usecase.WalletUseCase
}

func NewWalletConsumerHandler(
	logger log.Log,
	walletUsecase *usecase.WalletUseCase,
) *WalletConsumerHandler {
	return &WalletConsumerHandler{
		logger:        logger,
		WalletUseCase: walletUsecase,
	}
}

func (h *WalletConsumerHandler) HandleMessage(message *k.Message) {
	h.logger.Info(
		"wallet-consumer",
		fmt.Sprintf("Received message: %s", string(message.Value)),
		"HandleMessage",
		"",
	)

	var event model.NotificationUser
	if err := json.Unmarshal(message.Value, &event); err != nil {
		h.logger.Error(
			"wallet-consumer",
			fmt.Sprintf("Failed to unmarshal message: %v", err),
			"HandleMessage",
			"",
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := h.WalletUseCase.DebetWallet(ctx, &event)
	if err != nil {
		h.logger.Error(
			"wallet-consumer",
			fmt.Sprintf("Failed to handle wallet event: %v", err),
			"HandleMessage",
			"",
		)
		return
	}

	h.logger.Info(
		"wallet-consumer",
		fmt.Sprintf("Successfully processed wallet event: %+v", event),
		"HandleMessage",
		"",
	)
}

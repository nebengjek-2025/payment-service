package usecase

import (
	"context"
	"fmt"
	"notification-service/src/internal/gateway/messaging"
	"notification-service/src/internal/model"
	"notification-service/src/internal/repository"
	httpError "notification-service/src/pkg/http-error"
	"notification-service/src/pkg/log"
	"notification-service/src/pkg/utils"

	"github.com/redis/go-redis/v9"
)

type HandlePassengerEventRequest struct {
	PassengerID string
	Status      string
	RawMessage  string
}

type PassengerUseCase struct {
	log                    log.Log
	userRepository         *repository.UserRepository
	walletRepository       *repository.WalletRepository
	notificationRepository *repository.NotificationRepository
	Redis                  redis.UniversalClient
	UserProducer           *messaging.UserProducer
}

func NewPassengerUseCase(
	log log.Log,
	userRepo *repository.UserRepository,
	walletRepo *repository.WalletRepository,
	notifRepo *repository.NotificationRepository,
	redisClient redis.UniversalClient,
	userProducer *messaging.UserProducer,
) *PassengerUseCase {
	return &PassengerUseCase{
		log:                    log,
		userRepository:         userRepo,
		walletRepository:       walletRepo,
		notificationRepository: notifRepo,
		Redis:                  redisClient,
		UserProducer:           userProducer,
	}
}

func (uc *PassengerUseCase) GetInboxNotification(ctx context.Context, request *model.GetUserRequest) utils.Result {
	var result utils.Result

	userID := request.ID
	limit := 20
	offset := 0

	notifications, err := uc.notificationRepository.GetInboxNotifications(ctx, userID, limit, offset)
	if err != nil {
		errObj := httpError.NewInternalServerError()
		errObj.Message = fmt.Sprintf("Failed to get inbox notification: %v", err)
		result.Error = errObj
		uc.log.Error("passenger-usecase", errObj.Message, "GetInboxNotification", utils.ConvertString(err))
		return result
	}
	items := make([]model.NotificationItem, 0, len(notifications))
	for _, n := range notifications {
		item := model.NotificationItem{
			NotificationID: n.NotificationID,
			Title:          n.Title,
			Message:        n.Message,
			Type:           n.Type,
			OrderID:        n.OrderID,
			IsRead:         n.IsRead,
			Priority:       n.Priority,
			CreatedAt:      n.CreatedAt,
			ReadAt:         n.ReadAt,
		}
		items = append(items, item)
	}
	response := model.InboxNotificationResponse{
		UserID:        userID,
		Notifications: items,
	}

	result.Data = response
	return result
}

func (uc *PassengerUseCase) HandlePassengerEvent(ctx context.Context, req HandlePassengerEventRequest) error {

	uc.log.Info("passenger-usecase",
		"Processing passenger event from kafka",
		"HandlePassengerEvent",
		"")

	return nil
}

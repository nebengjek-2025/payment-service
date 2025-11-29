package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"payment-service/src/internal/entity"
	"payment-service/src/internal/model"
	"payment-service/src/internal/repository"
	httpError "payment-service/src/pkg/http-error"
	"payment-service/src/pkg/log"
	"payment-service/src/pkg/utils"
	"time"

	"github.com/redis/go-redis/v9"
)

type PassengerUseCase struct {
	log                    log.Log
	UserRepository         *repository.UserRepository
	NotificationRepository *repository.NotificationRepository
	OrderRepository        *repository.OrderRepository
	Redis                  redis.UniversalClient
}

func NewPassengerUseCase(
	log log.Log,
	userRepo *repository.UserRepository,
	notifRepo *repository.NotificationRepository,
	orderRepo *repository.OrderRepository,
	redisClient redis.UniversalClient,
) *PassengerUseCase {
	return &PassengerUseCase{
		log:                    log,
		UserRepository:         userRepo,
		NotificationRepository: notifRepo,
		OrderRepository:        orderRepo,
		Redis:                  redisClient,
	}
}

func (uc *PassengerUseCase) GetInboxNotification(ctx context.Context, request *model.GetUserRequest) utils.Result {
	var result utils.Result

	userID := request.ID
	limit := 20
	offset := 0

	notifications, err := uc.NotificationRepository.GetInboxNotifications(ctx, userID, limit, offset)
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

func (uc *PassengerUseCase) SendNotificationPassanger(ctx context.Context, req *model.NotificationUser) error {
	uc.log.Info(
		"passenger-usecase",
		fmt.Sprintf("Processing passenger notification from kafka: %+v", req),
		"SendNotificationPassanger",
		"",
	)

	if req == nil {
		return fmt.Errorf("request is nil")
	}
	passengerID := req.PassengerID
	if passengerID == "" {
		filter := entity.OrderFilter{
			OrderID: &req.OrderID,
		}

		order, err := uc.OrderRepository.FindOneOrder(ctx, filter)
		if err != nil {
			uc.log.Error(
				"passenger-usecase",
				fmt.Sprintf("Failed to find order when resolving passengerID: %v", err),
				"SendNotificationPassanger",
				utils.ConvertString(err),
			)
			return fmt.Errorf("failed to resolve passenger from order: %w", err)
		}

		if order == nil || order.PassengerID == "" {
			uc.log.Error(
				"passenger-usecase",
				fmt.Sprintf("Order not found or passengerID empty for order_id=%s", req.OrderID),
				"SendNotificationPassanger",
				"",
			)
			return fmt.Errorf("passenger not found for order_id=%s", req.OrderID)
		}

		passengerID = order.PassengerID
	}
	if req.OrderID == "" {
		return fmt.Errorf("order_id is required")
	}

	var (
		title    string
		message  string
		priority string
	)

	switch req.EventType {
	case "ORDER_MATCHING":
		title = "Sedang mencarikan driver untuk perjalanan Anda"
		message = fmt.Sprintf(
			"Permintaan perjalanan Anda dengan Order ID %s sedang kami proses.\n"+
				"Driver yang sesuai akan segera ditugaskan untuk menjemput Anda.",
			req.OrderID,
		)
		priority = "MEDIUM"

	case "DRIVER_ON_THE_WAY":
		title = "Driver dalam perjalanan menjemput Anda"
		message = fmt.Sprintf(
			"Driver telah ditugaskan dan sedang menuju titik penjemputan untuk Order ID %s.\n"+
				"Harap bersiap di lokasi penjemputan.",
			req.OrderID,
		)
		priority = "HIGH"

	default:
		uc.log.Error(
			"passenger-usecase",
			fmt.Sprintf("Unknown eventType for passenger notification: %s", req.EventType),
			"SendNotificationPassanger",
			"",
		)
		return nil
	}

	meta, err := json.Marshal(req)
	if err != nil {
		uc.log.Error(
			"passenger-usecase",
			fmt.Sprintf("Failed to marshal notification metadata: %v", err),
			"SendNotificationPassanger",
			"",
		)
		meta = nil
	}

	createdAt := req.Timestamp
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	notificationID := fmt.Sprintf("NTF_USR_%d", time.Now().UnixNano())

	notif := entity.Notification{
		NotificationID: notificationID,
		UserID:         passengerID,
		Title:          title,
		Message:        message,
		Type:           "ORDER_UPDATE",
		OrderID:        &req.OrderID,
		IsRead:         false,
		Priority:       priority,
		Metadata:       meta,
		CreatedAt:      createdAt,
	}

	if err := uc.NotificationRepository.SaveNotification(ctx, notif); err != nil {
		uc.log.Error(
			"passenger-usecase",
			fmt.Sprintf("Failed to save passenger notification: %v", err),
			"SendNotificationPassanger",
			utils.ConvertString(err),
		)
		return err
	}

	uc.log.Info(
		"passenger-usecase",
		fmt.Sprintf("Passenger notification saved successfully: notification_id=%s user_id=%s order_id=%s",
			notificationID,
			req.PassengerID,
			req.OrderID,
		),
		"SendNotificationPassanger",
		"",
	)

	// to do integration ke push notif fcm kalau ada budget

	return nil
}

func (uc *PassengerUseCase) SendNotificationOrder(ctx context.Context, evt *model.OrderNotificationEvent) error {
	if evt == nil {
		return fmt.Errorf("event is nil")
	}

	uc.log.Info(
		"passenger-usecase",
		fmt.Sprintf("Processing order notification event: %+v", evt),
		"SendNotificationOrder",
		"",
	)

	passengerID := evt.Message.PassengerID
	driverID := evt.Message.DriverID
	orderID := evt.Message.OrderID

	if orderID == "" {
		return fmt.Errorf("order_id is required")
	}
	if passengerID == "" {
		err := fmt.Errorf("passenger_id is required in message")
		uc.log.Error(
			"passenger-usecase",
			"PassengerID is empty in SendNotificationOrder",
			"SendNotificationOrder",
			utils.ConvertString(err),
		)
		return err
	}

	title := "Perjalanan Anda sedang berlangsung"
	message := fmt.Sprintf(
		"Perjalanan Anda dengan Order ID %s sedang berjalan bersama driver %s.\n"+
			"Silakan pastikan Anda sudah berada di kendaraan yang tepat dan gunakan fitur bantuan bila diperlukan.",
		orderID,
		driverID,
	)
	priority := "HIGH"

	meta, err := json.Marshal(evt)
	if err != nil {
		uc.log.Error(
			"passenger-usecase",
			fmt.Sprintf("Failed to marshal notification metadata in SendNotificationOrder: %v", err),
			"SendNotificationOrder",
			"",
		)
		meta = nil
	}

	notificationID := fmt.Sprintf("NTF_ORD_%d", time.Now().UnixNano())
	createdAt := time.Now()

	notif := entity.Notification{
		NotificationID: notificationID,
		UserID:         passengerID,
		Title:          title,
		Message:        message,
		Type:           "ORDER_UPDATE",
		OrderID:        &orderID,
		IsRead:         false,
		Priority:       priority,
		Metadata:       meta,
		CreatedAt:      createdAt,
	}

	if err := uc.NotificationRepository.SaveNotification(ctx, notif); err != nil {
		uc.log.Error(
			"passenger-usecase",
			fmt.Sprintf("Failed to save order notification: %v", err),
			"SendNotificationOrder",
			utils.ConvertString(err),
		)
		return fmt.Errorf("Failed to save order notification: %v", err)
	}

	uc.log.Info(
		"passenger-usecase",
		fmt.Sprintf("Order notification saved successfully: notification_id=%s user_id=%s order_id=%s",
			notificationID,
			passengerID,
			orderID,
		),
		"SendNotificationOrder",
		"",
	)

	// TODO: nanti ada push notif (FCM)

	return nil
}

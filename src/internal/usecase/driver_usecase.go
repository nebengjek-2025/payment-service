package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"payment-service/src/pkg/log"
	"time"

	"payment-service/src/internal/entity"
	"payment-service/src/internal/model"
	"payment-service/src/internal/repository"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

type DriverUseCase struct {
	Log                    log.Log
	UserRepository         *repository.UserRepository
	OrderRepository        *repository.OrderRepository
	DriverRepository       *repository.DriverRepository
	NotificationRepository *repository.NotificationRepository
	Config                 *viper.Viper
	Redis                  redis.UniversalClient
}

func NewDriverUseCase(
	logger log.Log,
	userRepository *repository.UserRepository,
	driverRepository *repository.DriverRepository,
	orderRepository *repository.OrderRepository,
	notificationRepo *repository.NotificationRepository,
	redisClient redis.UniversalClient,
) *DriverUseCase {
	return &DriverUseCase{
		Log:                    logger,
		UserRepository:         userRepository,
		DriverRepository:       driverRepository,
		OrderRepository:        orderRepository,
		NotificationRepository: notificationRepo,
		Redis:                  redisClient,
	}
}

func (d *DriverUseCase) SendNotificationDriver(ctx context.Context, req model.DriverEvent) error {
	d.Log.Info(
		"driver-usecase",
		fmt.Sprintf("SendNotificationDriver called with payload: %+v", req),
		"SendNotificationDriver",
		"",
	)
	var (
		originAddr string
		destAddr   string
	)

	if req.RouteSummary.Route.Origin.Address != "" {
		originAddr = req.RouteSummary.Route.Origin.Address
	}
	if req.RouteSummary.Route.Destination.Address != "" {
		destAddr = req.RouteSummary.Route.Destination.Address
	}

	const (
		NotificationTypeDriverAssignment = "DRIVER_ASSIGNMENT"
		NotificationPriorityHigh         = "HIGH"
	)

	title := "Order Baru Penumpang memilih Anda"
	message := fmt.Sprintf(
		"Anda menerima permintaan perjalanan baru.\n\nOrder ID: %s\nPickup: %s\nTujuan: %s\nJarak: %.2f km\nDurasi: %s\nPerkiraan harga: Rp%.0f - Rp%.0f",
		req.OrderID,
		originAddr,
		destAddr,
		req.RouteSummary.BestRouteKm,
		req.RouteSummary.BestRouteDuration,
		req.RouteSummary.MinPrice,
		req.RouteSummary.MaxPrice,
	)

	meta, err := json.Marshal(req.RouteSummary)
	if err != nil {
		d.Log.Error(
			"driver-usecase",
			fmt.Sprintf("Failed to marshal route_summary: %v", err),
			"SendNotificationDriver",
			"",
		)
	}

	notif := entity.Notification{
		NotificationID: fmt.Sprintf("NTF_DRV_%d", time.Now().UnixNano()),
		UserID:         req.DriverID,
		Title:          title,
		Message:        message,
		Type:           NotificationTypeDriverAssignment,
		OrderID:        &req.OrderID,
		IsRead:         false,
		Priority:       NotificationPriorityHigh,
		Metadata:       meta,
		CreatedAt:      time.Now(),
	}
	if err := d.NotificationRepository.SaveNotification(ctx, notif); err != nil {
		d.Log.Error(
			"driver-usecase",
			fmt.Sprintf("Failed to save notification: %v", err),
			"SendNotificationDriver",
			"",
		)
		return err
	}

	return nil
}

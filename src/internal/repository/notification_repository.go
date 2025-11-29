package repository

import (
	"context"
	"payment-service/src/internal/entity"
	"payment-service/src/pkg/databases/mysql"
)

type NotificationRepository struct {
	DB mysql.DBInterface
}

func NewNotificationRepository(db mysql.DBInterface) *NotificationRepository {
	return &NotificationRepository{DB: db}
}

func (r *NotificationRepository) GetInboxNotifications(ctx context.Context, userID string, limit, offset int) ([]entity.Notification, error) {
	db, err := r.DB.GetDB()
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT 
			id,
			notification_id,
			user_id,
			title,
			message,
			type,
			order_id,
			is_read,
			priority,
			metadata,
			created_at,
			read_at
		FROM notifications
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	var notifications []entity.Notification
	if err := db.SelectContext(ctx, &notifications, query, userID, limit, offset); err != nil {
		return nil, err
	}

	return notifications, nil
}

func (r *NotificationRepository) SaveNotification(ctx context.Context, notif entity.Notification) error {
	db, err := r.DB.GetDB()
	if err != nil {
		return err
	}

	query := `
		INSERT INTO notifications (
			notification_id,
			user_id,
			title,
			message,
			type,
			order_id,
			is_read,
			priority,
			metadata,
			created_at,
			read_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	// pastikan pointer field aman
	var orderID any
	if notif.OrderID != nil {
		orderID = *notif.OrderID
	} else {
		orderID = nil
	}

	var readAt any
	if notif.ReadAt != nil {
		readAt = *notif.ReadAt
	} else {
		readAt = nil
	}

	_, err = db.ExecContext(
		ctx,
		query,
		notif.NotificationID,
		notif.UserID,
		notif.Title,
		notif.Message,
		notif.Type,
		orderID,
		notif.IsRead,
		notif.Priority,
		notif.Metadata,
		notif.CreatedAt,
		readAt,
	)
	if err != nil {
		return err
	}

	return nil
}

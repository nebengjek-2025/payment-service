package repository

import (
	"context"
	"notification-service/src/internal/entity"
	"notification-service/src/pkg/databases/mysql"
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

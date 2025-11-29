package entity

import "time"

type Notification struct {
	ID             uint64     `db:"id"              json:"id"`
	NotificationID string     `db:"notification_id" json:"notification_id"`
	UserID         string     `db:"user_id"         json:"user_id"`
	Title          string     `db:"title"           json:"title"`
	Message        string     `db:"message"         json:"message"`
	Type           string     `db:"type"            json:"type"`
	OrderID        *string    `db:"order_id"        json:"order_id,omitempty"`
	IsRead         bool       `db:"is_read"         json:"is_read"`
	Priority       string     `db:"priority"        json:"priority"`
	Metadata       []byte     `db:"metadata"        json:"-"`
	CreatedAt      time.Time  `db:"created_at"      json:"created_at"`
	ReadAt         *time.Time `db:"read_at"        json:"read_at,omitempty"`
}

package model

import "time"

type UserEvent struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	CreatedAt int64  `json:"created_at,omitempty"`
	UpdatedAt int64  `json:"updated_at,omitempty"`
}

func (u *UserEvent) GetId() string {
	return u.ID
}

type NotificationUser struct {
	EventType   string    `json:"eventType"`
	OrderID     string    `json:"orderId"`
	DriverID    string    `json:"driverId"`
	PassengerID string    `json:"passangerId"`
	Timestamp   time.Time `json:"timestamp"`
}

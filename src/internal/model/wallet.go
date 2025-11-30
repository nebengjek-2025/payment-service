package model

import "time"

type WalletRequest struct {
	UserID string `json:"userId"`
	Amount int64  `json:"amount"`
}

type WalletTransactionHistory struct {
	TransactionID string    `json:"transaction_id"`
	Amount        float64   `json:"amount"`
	Type          string    `json:"type"` // credit / debit
	Description   string    `json:"description"`
	Timestamp     time.Time `json:"timestamp"`
}

type WalletResponse struct {
	UserID       string                     `json:"user_id"`
	Balance      float64                    `json:"balance"`
	Transactions []WalletTransactionHistory `json:"transactions"`
}

type WalletHoldRequest struct {
	OrderID     string `json:"orderId" validate:"required"`
	PassengerID string `json:"passengerId" validate:"required"`
	DriverID    string `json:"driverId" validate:"required"`
}

type OrderNotificationMessage struct {
	DriverID    string `json:"driverId"`
	PassengerID string `json:"passangerId"`
	OrderID     string `json:"orderId"`
}

type OrderNotificationEvent struct {
	ID      string                   `json:"id"`
	Message OrderNotificationMessage `json:"message"`
}

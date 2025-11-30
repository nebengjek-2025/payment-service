package model

import "time"

type UserResponse struct {
	ID           string     `json:"id,omitempty"`
	Name         string     `json:"name,omitempty"`
	MobileNumber string     `json:"mobile_number,omitempty"`
	CreatedAt    time.Time  `json:"created_at,omitempty"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
}

type GetUserRequest struct {
	ID string `json:"id" validate:"required,max=100"`
}

type FindDriverRequest struct {
	UserID        string `json:"userId" validate:"required"`
	PaymentMethod string `json:"paymentMethod" validate:"required,oneof=wallet cash qris"`
}

type LocationRequest struct {
	Longitude float64 `json:"longitude" validate:"required"`
	Latitude  float64 `json:"latitude" validate:"required"`
	Address   string  `json:"address" validate:"required"`
}

type LocationSuggestionRequest struct {
	CurrentLocation LocationRequest `json:"currentLocation" validate:"required"`
	Destination     LocationRequest `json:"destination" validate:"required"`
	UserID          string          `json:"userId" validate:"required"`
}

type RouteSummary struct {
	Route             Route   `json:"route"`
	MinPrice          float64 `json:"minPrice"`
	MaxPrice          float64 `json:"maxPrice"`
	BestRouteKm       float64 `json:"bestRouteKm"`
	BestRoutePrice    float64 `json:"bestRoutePrice"`
	BestRouteDuration string  `json:"bestRouteDuration"`
	Duration          int     `json:"duration"`
}

type RequestRide struct {
	RouteSummary RouteSummary `json:"routeSummary" bson:"routeSummary"`
	UserId       string       `json:"userId" bson:"userId"`
}

type Route struct {
	Origin      LocationRequest `json:"origin" `
	Destination LocationRequest `json:"destination"`
}

type FindDriverResponse struct {
	Message string      `json:"message"`
	Driver  interface{} `json:"driver"`
}

type Wallet struct {
	ID          string    `db:"id"        json:"id"`
	UserID      string    `db:"user_id"   json:"user_id"`
	Balance     float64   `db:"balance"   json:"balance"`
	LastUpdated time.Time `db:"last_updated" json:"last_updated"`
	CreatedAt   time.Time `db:"created_at"   json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"   json:"updated_at"`
}

type WalletTransaction struct {
	ID            uint64    `db:"id"             json:"id"`
	WalletID      string    `db:"wallet_id"      json:"wallet_id"`
	TransactionID string    `db:"transaction_id" json:"transaction_id"`
	Amount        float64   `db:"amount"         json:"amount"`
	Type          string    `db:"type"           json:"type"` // "credit" / "debit"
	Description   string    `db:"description"    json:"description"`
	Timestamp     time.Time `db:"timestamp"      json:"timestamp"`
	CreatedAt     time.Time `db:"created_at"     json:"created_at"`
}

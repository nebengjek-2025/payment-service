package entity

import (
	"time"
)

type PaymentTransaction struct {
	ID                  uint64     `db:"id"`
	RideOrderID         uint64     `db:"ride_order_id"`
	PassengerID         string     `db:"passenger_id"`
	DriverID            string     `db:"driver_id"`
	Amount              float64    `db:"amount"`
	Currency            string     `db:"currency"`
	PaymentMethod       string     `db:"payment_method"`
	PaymentStatus       string     `db:"payment_status"`
	ProviderName        *string    `db:"provider_name"`
	ProviderReferenceID *string    `db:"provider_reference_id"`
	ProviderSignature   *string    `db:"provider_signature"`
	PaidAt              *time.Time `db:"paid_at"`
	ExpiredAt           *time.Time `db:"expired_at"`
	Metadata            []byte     `db:"metadata"`
	CreatedAt           time.Time  `db:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at"`
}

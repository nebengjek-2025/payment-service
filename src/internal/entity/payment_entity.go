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
	RefundedAt          *time.Time `db:"refunded_at"`
	Metadata            []byte     `db:"metadata"`
	CreatedAt           time.Time  `db:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at"`
}

type PaymentEventLog struct {
	ID                   uint64    `db:"id" json:"id"`
	PaymentTransactionID uint64    `db:"payment_transaction_id" json:"payment_transaction_id"`
	EventType            string    `db:"event_type" json:"event_type"`
	EventDescription     string    `db:"event_description" json:"event_description,omitempty"`
	RawPayload           *string   `db:"raw_payload" json:"raw_payload,omitempty"`
	CreatedAt            time.Time `db:"created_at" json:"created_at"`
}

type PaymentSettlement struct {
	ID                   uint64     `db:"id"`
	PaymentTransactionID uint64     `db:"payment_transaction_id"`
	DriverID             string     `db:"driver_id"`
	SettlementAmount     float64    `db:"settlement_amount"`
	PlatformFee          float64    `db:"platform_fee"`
	TaxAmount            float64    `db:"tax_amount"`
	Status               string     `db:"status"`
	SettlementMethod     string     `db:"settlement_method"`
	ProviderReferenceID  *string    `db:"provider_reference_id"`
	SettledAt            *time.Time `db:"settled_at"`
	Metadata             *string    `db:"metadata"`
	CreatedAt            time.Time  `db:"created_at"`
	UpdatedAt            time.Time  `db:"updated_at"`
}

package model

type CreateQrisPaymentRequest struct {
	OrderID string `json:"orderId" validate:"required"`
	UserID  string `json:"userId" validate:"required"`
}

type QrisSnapPaymentResponse struct {
	OrderID       string  `json:"order_id"`
	Amount        float64 `json:"amount"`
	SnapToken     string  `json:"snap_token"`
	RedirectURL   string  `json:"redirect_url"`
	TransactionID string  `json:"transaction_id,omitempty"`
	Status        string  `json:"status,omitempty"`
}

type QrisPaymentResponse struct {
	OrderID            string  `json:"order_id"`
	Amount             float64 `json:"amount"`
	PaymentURL         string  `json:"payment_url,omitempty"`
	QrString           string  `json:"qr_string,omitempty"`
	TransactionID      string  `json:"transaction_id"`
	TransactionStatus  string  `json:"transaction_status"`
	ExpiryTime         string  `json:"expiry_time,omitempty"`
	PaymentProviderRef string  `json:"payment_provider_ref,omitempty"`
}

type MidtransNotification struct {
	TransactionTime   string `json:"transaction_time"`
	TransactionStatus string `json:"transaction_status"`
	TransactionID     string `json:"transaction_id"`
	StatusMessage     string `json:"status_message"`
	StatusCode        string `json:"status_code"`
	SignatureKey      string `json:"signature_key"`
	SettlementTime    string `json:"settlement_time"`
	PaymentType       string `json:"payment_type"`
	OrderID           string `json:"order_id"`
	MerchantID        string `json:"merchant_id"`
	GrossAmount       string `json:"gross_amount"`
	FraudStatus       string `json:"fraud_status"`
}

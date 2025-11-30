package repository

import (
	"context"
	"payment-service/src/internal/entity"
	"payment-service/src/pkg/databases/mysql"

	"github.com/jmoiron/sqlx"
)

type PaymentRepository struct {
	DB mysql.DBInterface
}

func NewPaymentRepository(db mysql.DBInterface) *PaymentRepository {
	return &PaymentRepository{DB: db}
}

func (r *PaymentRepository) InsertPaymentTransactionTx(
	ctx context.Context,
	tx *sqlx.Tx,
	p *entity.PaymentTransaction,
) (uint64, error) {
	query := `
		INSERT INTO payment_transactions (
			ride_order_id,
			passenger_id,
			driver_id,
			amount,
			currency,
			payment_method,
			payment_status,
			provider_name,
			provider_reference_id,
			provider_signature,
			paid_at,
			expired_at,
			metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := tx.ExecContext(ctx, query,
		p.RideOrderID,
		p.PassengerID,
		p.DriverID,
		p.Amount,
		p.Currency,
		p.PaymentMethod,
		p.PaymentStatus,
		p.ProviderName,
		p.ProviderReferenceID,
		p.ProviderSignature,
		p.PaidAt,
		p.ExpiredAt,
		p.Metadata,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint64(id), err
}

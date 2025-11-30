package repository

import (
	"context"
	"database/sql"
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

func (r *PaymentRepository) FindPendingPaymentByOrder(ctx context.Context, tx *sql.Tx, orderID uint64) (*entity.PaymentTransaction, error) {
	var payment entity.PaymentTransaction
	query := `
		SELECT 
			id,
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
			refunded_at,
			metadata,
			created_at,
			updated_at
		FROM payment_transactions
		WHERE ride_order_id = ? AND payment_status = 'PENDING'
		LIMIT 1;
	`

	var err error
	if tx != nil {
		err = tx.QueryRowContext(ctx, query, orderID).Scan(
			&payment.ID,
			&payment.RideOrderID,
			&payment.PassengerID,
			&payment.DriverID,
			&payment.Amount,
			&payment.Currency,
			&payment.PaymentMethod,
			&payment.PaymentStatus,
			&payment.ProviderName,
			&payment.ProviderReferenceID,
			&payment.ProviderSignature,
			&payment.PaidAt,
			&payment.ExpiredAt,
			&payment.RefundedAt,
			&payment.Metadata,
			&payment.CreatedAt,
			&payment.UpdatedAt,
		)
	} else {
		db, err := r.DB.GetDB()
		if err != nil {
			return nil, err
		}
		err = db.QueryRowContext(ctx, query, orderID).Scan(
			&payment.ID,
			&payment.RideOrderID,
			&payment.PassengerID,
			&payment.DriverID,
			&payment.Amount,
			&payment.Currency,
			&payment.PaymentMethod,
			&payment.PaymentStatus,
			&payment.ProviderName,
			&payment.ProviderReferenceID,
			&payment.ProviderSignature,
			&payment.PaidAt,
			&payment.ExpiredAt,
			&payment.RefundedAt,
			&payment.Metadata,
			&payment.CreatedAt,
			&payment.UpdatedAt,
		)
	}

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &payment, nil
}

func (r *PaymentRepository) InsertPaymentEventLogTx(ctx context.Context, tx *sql.Tx, log *entity.PaymentEventLog) error {
	query := `
		INSERT INTO payment_event_logs (
			payment_transaction_id,
			event_type,
			event_description,
			raw_payload
		) VALUES (?, ?, ?, ?)
	`

	_, err := tx.ExecContext(ctx, query,
		log.PaymentTransactionID,
		log.EventType,
		log.EventDescription,
		log.RawPayload,
	)
	return err
}

func (r *PaymentRepository) UpdatePaymentTransactionTx(
	ctx context.Context,
	tx *sqlx.Tx,
	p *entity.PaymentTransaction,
) error {
	query := `
		UPDATE payment_transactions
		SET
			amount              = ?,
			currency            = ?,
			payment_method      = ?,
			payment_status      = ?,
			provider_name       = ?,
			provider_reference_id = ?,
			provider_signature  = ?,
			paid_at             = ?,
			expired_at          = ?,
			refunded_at         = ?,
			metadata            = ?,
			updated_at          = NOW()
		WHERE id = ?
	`

	_, err := tx.ExecContext(
		ctx,
		query,
		p.Amount,
		p.Currency,
		p.PaymentMethod,
		p.PaymentStatus,
		p.ProviderName,
		p.ProviderReferenceID,
		p.ProviderSignature,
		p.PaidAt,
		p.ExpiredAt,
		p.RefundedAt,
		p.Metadata,
		p.ID,
	)
	return err
}

func (r *OrderRepository) MarkOrderPaidTx(ctx context.Context, tx *sql.Tx, orderID, passengerID, driverID string) (bool, error) {
	query := `
		UPDATE orders
		SET 
			payment_status = 'PAID',
			status = 'COMPLETED',
			updated_at = NOW()
		WHERE 
			order_id = ?
			AND passenger_id = ?
			AND driver_id = ?
			AND status = 'COMPLETED'
			AND payment_status = 'UNPAID'
	`

	res, err := tx.ExecContext(ctx, query, orderID, passengerID, driverID)
	if err != nil {
		return false, err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return rows > 0, nil
}

func (r *PaymentRepository) FindByOrderIDForUpdate(ctx context.Context, tx *sqlx.Tx, orderID string) (*entity.PaymentTransaction, error) {
	query := `
		SELECT *
		FROM payment_transactions
		WHERE provider_reference_id = ? OR ride_order_id = (
			SELECT id FROM orders WHERE order_id = ?
		)
		LIMIT 1
		FOR UPDATE
	`

	var p entity.PaymentTransaction
	err := tx.GetContext(ctx, &p, query, orderID, orderID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PaymentRepository) InsertPaymentSettlementTx(
	ctx context.Context,
	tx *sqlx.Tx,
	s *entity.PaymentSettlement,
) error {
	query := `
		INSERT INTO payment_settlements (
			payment_transaction_id,
			driver_id,
			settlement_amount,
			platform_fee,
			tax_amount,
			status,
			settlement_method,
			provider_reference_id,
			settled_at,
			metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	res, err := tx.ExecContext(
		ctx,
		query,
		s.PaymentTransactionID,
		s.DriverID,
		s.SettlementAmount,
		s.PlatformFee,
		s.TaxAmount,
		s.Status,
		s.SettlementMethod,
		s.ProviderReferenceID,
		s.SettledAt,
		s.Metadata,
	)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	s.ID = uint64(id)
	return nil
}

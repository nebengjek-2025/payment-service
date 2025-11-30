package repository

import (
	"context"
	"database/sql"
	"payment-service/src/internal/entity"
	"payment-service/src/pkg/databases/mysql"
)

type WalletRepository struct {
	DB mysql.DBInterface
}

func NewWalletRepository(db mysql.DBInterface) *WalletRepository {
	return &WalletRepository{DB: db}
}

func (r *WalletRepository) GetWalletByUserID(ctx context.Context, userID string) (*entity.Wallet, error) {
	db, err := r.DB.GetDB()
	if err != nil {
		return nil, err
	}

	var w entity.Wallet
	query := `
		SELECT id, user_id, balance, last_updated, created_at, updated_at
		FROM wallets
		WHERE user_id = ?
		LIMIT 1
	`
	if err := db.GetContext(ctx, &w, query, userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &w, nil
}

func (r *WalletRepository) GetWalletForUpdate(ctx context.Context, tx *sql.Tx, userID string) (*entity.Wallet, error) {
	var w entity.Wallet
	query := `
		SELECT id, user_id, balance, last_updated, created_at, updated_at
		FROM wallets
		WHERE user_id = ?
		FOR UPDATE
	`
	err := tx.QueryRowContext(ctx, query, userID).Scan(
		&w.ID, &w.UserID, &w.Balance, &w.LastUpdated, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &w, nil
}

func (r *WalletRepository) InsertWallet(ctx context.Context, tx *sql.Tx, w *entity.Wallet) error {
	query := `
		INSERT INTO wallets (id, user_id, balance, last_updated, created_at, updated_at)
		VALUES (?, ?, ?, NOW(6), NOW(6), NOW(6))
	`
	_, err := tx.ExecContext(ctx, query, w.ID, w.UserID, w.Balance)
	return err
}

func (r *WalletRepository) UpdateWalletBalance(ctx context.Context, tx *sql.Tx, walletID string, newBalance float64) error {
	query := `
		UPDATE wallets
		SET balance = ?, last_updated = NOW(6)
		WHERE id = ?
	`
	_, err := tx.ExecContext(ctx, query, newBalance, walletID)
	return err
}

func (r *WalletRepository) InsertWalletTransaction(ctx context.Context, tx *sql.Tx, trx *entity.WalletTransaction) error {
	query := `
		INSERT INTO wallet_transactions (
			wallet_id, transaction_id, amount, type, description, timestamp, created_at
		) VALUES (?, ?, ?, ?, ?, NOW(6), NOW(6))
	`
	_, err := tx.ExecContext(ctx, query,
		trx.WalletID, trx.TransactionID, trx.Amount, trx.Type, trx.Description,
	)
	return err
}

func (r *WalletRepository) GetLastTransactions(ctx context.Context, walletID string, limit int) ([]entity.WalletTransaction, error) {
	db, err := r.DB.GetDB()
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 5
	}

	query := `
		SELECT 
			id, wallet_id, transaction_id, amount, type, description, timestamp, created_at
		FROM wallet_transactions
		WHERE wallet_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	var txs []entity.WalletTransaction
	if err := db.SelectContext(ctx, &txs, query, walletID, limit); err != nil {
		return nil, err
	}
	return txs, nil
}

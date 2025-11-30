package usecase

import (
	"context"
	"fmt"
	"payment-service/src/internal/entity"
	"payment-service/src/internal/model"
	"payment-service/src/internal/repository"
	"payment-service/src/pkg/databases/mysql"
	httpError "payment-service/src/pkg/http-error"
	"payment-service/src/pkg/log"
	"payment-service/src/pkg/utils"
	"time"

	"github.com/redis/go-redis/v9"
)

type WalletUseCase struct {
	Log               log.Log
	UserRepository    *repository.UserRepository
	WalletRepository  *repository.WalletRepository
	PaymentRepository *repository.PaymentRepository
	OrderRepository   *repository.OrderRepository
	DB                mysql.DBInterface
	Redis             redis.UniversalClient
}

func NewWalletUseCase(
	log log.Log,
	userRepo *repository.UserRepository,
	orderRepo *repository.OrderRepository,
	walletRepo *repository.WalletRepository,
	paymentRepo *repository.PaymentRepository,
	db mysql.DBInterface,
	redisClient redis.UniversalClient,
) *WalletUseCase {
	return &WalletUseCase{
		Log:               log,
		UserRepository:    userRepo,
		WalletRepository:  walletRepo,
		PaymentRepository: paymentRepo,
		OrderRepository:   orderRepo,
		DB:                db,
		Redis:             redisClient,
	}
}

func (uc *WalletUseCase) TopUpWallet(ctx context.Context, request *model.WalletRequest) utils.Result {
	var result utils.Result

	if request.UserID == "" || request.Amount <= 0 {
		errObj := httpError.NewBadRequest()
		errObj.Message = "userId and positive amount are required"
		result.Error = errObj
		uc.Log.Error("wallet-usecase", errObj.Message, "TopUpWallet", utils.ConvertString(request))
		return result
	}

	db, err := uc.DB.GetDB()
	if err != nil {
		errObj := httpError.NewInternalServerError()
		errObj.Message = "failed to get db connection"
		result.Error = errObj
		uc.Log.Error("wallet-usecase", errObj.Message, "TopUpWallet", utils.ConvertString(err))
		return result
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		errObj := httpError.NewInternalServerError()
		errObj.Message = "failed to start transaction"
		result.Error = errObj
		uc.Log.Error("wallet-usecase", errObj.Message, "TopUpWallet", utils.ConvertString(err))
		return result
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	wallet, err := uc.WalletRepository.GetWalletForUpdate(ctx, tx.Tx, request.UserID)
	if err != nil {
		_ = tx.Rollback()
		errObj := httpError.NewInternalServerError()
		errObj.Message = "failed to get wallet"
		result.Error = errObj
		uc.Log.Error("wallet-usecase", errObj.Message, "TopUpWallet", utils.ConvertString(err))
		return result
	}

	if wallet == nil {
		walletID := utils.GenerateUniqueIDWithPrefix("wlt")
		wallet = &entity.Wallet{
			ID:      walletID,
			UserID:  request.UserID,
			Balance: 0,
		}
		if err := uc.WalletRepository.InsertWallet(ctx, tx.Tx, wallet); err != nil {
			_ = tx.Rollback()
			errObj := httpError.NewInternalServerError()
			errObj.Message = "failed to create wallet"
			result.Error = errObj
			uc.Log.Error("wallet-usecase", errObj.Message, "TopUpWallet", utils.ConvertString(err))
			return result
		}
	}

	amount := float64(request.Amount)
	newBalance := wallet.Balance + amount

	if err := uc.WalletRepository.UpdateWalletBalance(ctx, tx.Tx, wallet.ID, newBalance); err != nil {
		_ = tx.Rollback()
		errObj := httpError.NewInternalServerError()
		errObj.Message = "failed to update wallet balance"
		result.Error = errObj
		uc.Log.Error("wallet-usecase", errObj.Message, "TopUpWallet", utils.ConvertString(err))
		return result
	}

	trxID := utils.GenerateUniqueIDWithPrefix("wtrx")
	trx := &entity.WalletTransaction{
		WalletID:      wallet.ID,
		TransactionID: trxID,
		Amount:        amount,
		Type:          "credit",
		Description:   "Top up wallet",
		Timestamp:     time.Now(),
	}

	if err := uc.WalletRepository.InsertWalletTransaction(ctx, tx.Tx, trx); err != nil {
		_ = tx.Rollback()
		errObj := httpError.NewInternalServerError()
		errObj.Message = "failed to insert wallet transaction"
		result.Error = errObj
		uc.Log.Error("wallet-usecase", errObj.Message, "TopUpWallet", utils.ConvertString(err))
		return result
	}

	if err := tx.Commit(); err != nil {
		errObj := httpError.NewInternalServerError()
		errObj.Message = "failed to commit transaction"
		result.Error = errObj
		uc.Log.Error("wallet-usecase", errObj.Message, "TopUpWallet", utils.ConvertString(err))
		return result
	}

	// 6. Response
	result.Data = model.WalletResponse{
		UserID:  request.UserID,
		Balance: newBalance,
		Transactions: []model.WalletTransactionHistory{
			{
				TransactionID: trx.TransactionID,
				Amount:        trx.Amount,
				Type:          trx.Type,
				Description:   trx.Description,
				Timestamp:     trx.Timestamp,
			},
		},
	}

	return result
}

func (uc *WalletUseCase) GetWallet(ctx context.Context, request *model.GetUserRequest) utils.Result {
	var result utils.Result

	userID := request.ID

	wallet, err := uc.WalletRepository.GetWalletByUserID(ctx, userID)
	if err != nil {
		errObj := httpError.NewInternalServerError()
		errObj.Message = "failed to get wallet"
		result.Error = errObj
		uc.Log.Error("wallet-usecase", errObj.Message, "GetWallet", utils.ConvertString(err))
		return result
	}

	if wallet == nil {
		result.Data = model.WalletResponse{
			UserID:       userID,
			Balance:      0,
			Transactions: []model.WalletTransactionHistory{},
		}
		return result
	}

	txs, err := uc.WalletRepository.GetLastTransactions(ctx, wallet.ID, 5)
	if err != nil {
		errObj := httpError.NewInternalServerError()
		errObj.Message = "failed to get wallet transactions"
		result.Error = errObj
		uc.Log.Error("wallet-usecase", errObj.Message, "GetWallet", utils.ConvertString(err))
		return result
	}

	histories := make([]model.WalletTransactionHistory, 0, len(txs))
	for _, t := range txs {
		histories = append(histories, model.WalletTransactionHistory{
			TransactionID: t.TransactionID,
			Amount:        t.Amount,
			Type:          t.Type,
			Description:   t.Description,
			Timestamp:     t.Timestamp,
		})
	}

	result.Data = model.WalletResponse{
		UserID:       userID,
		Balance:      wallet.Balance,
		Transactions: histories,
	}

	return result
}

func (uc *WalletUseCase) HoldWalletForOrder(ctx context.Context, request *model.OrderNotificationEvent) error {
	order, err := uc.OrderRepository.FindOneOrder(ctx, entity.OrderFilter{OrderID: &request.ID, PassengerID: &request.Message.PassengerID, DriverID: &request.Message.DriverID})
	if err != nil || order == nil {
		uc.Log.Error("wallet-usecase", "Order not found", "HoldWalletForOrder", utils.ConvertString(err))
		return fmt.Errorf("order not found")
	}
	if order.PaymentMethod != "WALLET" && order.PaymentMethod != "EWALLET" {
		uc.Log.Error("wallet-usecase", "Payment method is not wallet", "HoldWalletForOrder", order.PaymentMethod)
		return fmt.Errorf("payment method is not wallet")
	}
	if order.PaymentStatus == "PAID" {
		uc.Log.Error("wallet-usecase", "Order is already paid", "HoldWalletForOrder", "")
		return fmt.Errorf("order is already paid")
	}
	amount := order.MaxPrice
	if amount <= 0 {
		uc.Log.Error("wallet-usecase", "Invalid order amount", "HoldWalletForOrder", utils.ConvertString(order))
		return fmt.Errorf("invalid order amount")
	}
	db, err := uc.DB.GetDB()
	if err != nil {
		uc.Log.Error("wallet-usecase", "failed to get db connection", "HoldWalletForOrder", utils.ConvertString(err))
		return fmt.Errorf("failed to get db connection, error : %v", err)
	}
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		uc.Log.Error("wallet-usecase", "failed to start transaction", "HoldWalletForOrder", utils.ConvertString(err))
		return fmt.Errorf("failed to start transaction, error : %v", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	wallet, err := uc.WalletRepository.GetWalletForUpdate(ctx, tx.Tx, request.Message.PassengerID)
	if err != nil {
		_ = tx.Rollback()
		uc.Log.Error("wallet-usecase", "failed to get wallet", "HoldWalletForOrder", utils.ConvertString(err))
		return fmt.Errorf("failed to get wallet, error : %v", err)
	}
	if wallet == nil {
		_ = tx.Rollback()
		uc.Log.Error("wallet-usecase", "Wallet not found for passenger", "HoldWalletForOrder", request.Message.PassengerID)
		return fmt.Errorf("wallet not found for passenger")
	}
	if wallet.Balance < amount {
		_ = tx.Rollback()
		uc.Log.Error("wallet-usecase", "Insufficient wallet balance", "HoldWalletForOrder",
			fmt.Sprintf("balance=%.2f need=%.2f", wallet.Balance, amount))
		return fmt.Errorf("balance=%.2f need=%.2f", wallet.Balance, amount)
	}
	newBalance := wallet.Balance - amount
	if err := uc.WalletRepository.UpdateWalletBalance(ctx, tx.Tx, wallet.ID, newBalance); err != nil {
		_ = tx.Rollback()
		uc.Log.Error("wallet-usecase", "failed to update wallet balance", "HoldWalletForOrder", utils.ConvertString(err))
		return fmt.Errorf("failed to update wallet balance, err:%v", err)
	}
	walletTrx := &entity.WalletTransaction{
		WalletID:      wallet.ID,
		TransactionID: utils.GenerateUniqueIDWithPrefix("wtrx"),
		Amount:        amount,
		Type:          "debit",
		Description:   fmt.Sprintf("Hold for order %s", request.Message.OrderID),
	}
	if err := uc.WalletRepository.InsertWalletTransaction(ctx, tx.Tx, walletTrx); err != nil {
		_ = tx.Rollback()
		uc.Log.Error("wallet-usecase", "failed to insert wallet transaction", "HoldWalletForOrder", utils.ConvertString(err))
		return err
	}
	payment := &entity.PaymentTransaction{
		RideOrderID:   order.ID,
		PassengerID:   request.Message.PassengerID,
		DriverID:      request.Message.DriverID,
		Amount:        amount,
		Currency:      "IDR",
		PaymentMethod: "EWALLET",
		PaymentStatus: "PENDING",
	}
	paymentID, err := uc.PaymentRepository.InsertPaymentTransactionTx(ctx, tx, payment)
	if err != nil {
		_ = tx.Rollback()
		uc.Log.Error("wallet-usecase", "failed to create payment transaction", "HoldWalletForOrder", utils.ConvertString(err))
		return err
	}

	if err := tx.Commit(); err != nil {
		uc.Log.Error("wallet-usecase", "failed to commit transaction", "HoldWalletForOrder", utils.ConvertString(err))
		return err
	}
	result := map[string]interface{}{
		"order_id":       request.ID,
		"passenger_id":   request.Message.PassengerID,
		"driver_id":      request.Message.DriverID,
		"amount_hold":    amount,
		"wallet_balance": newBalance,
		"payment_tx_id":  paymentID,
		"payment_status": "PENDING", // HOLD
		"message":        "Wallet balance held for this order",
	}
	uc.Log.Info("wallet-usecase", fmt.Sprintf("Sukses hold wallet %s status order pending", request.Message.OrderID), "HoldWalletForOrder", utils.ConvertString(result))

	return nil

}

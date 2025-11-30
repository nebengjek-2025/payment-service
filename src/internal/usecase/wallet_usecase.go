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
	"github.com/spf13/viper"
)

type WalletUseCase struct {
	Log               log.Log
	Config            *viper.Viper
	UserRepository    *repository.UserRepository
	WalletRepository  *repository.WalletRepository
	PaymentRepository *repository.PaymentRepository
	OrderRepository   *repository.OrderRepository
	DB                mysql.DBInterface
	Redis             redis.UniversalClient
}

func NewWalletUseCase(
	log log.Log,
	config *viper.Viper,
	userRepo *repository.UserRepository,
	orderRepo *repository.OrderRepository,
	walletRepo *repository.WalletRepository,
	paymentRepo *repository.PaymentRepository,
	db mysql.DBInterface,
	redisClient redis.UniversalClient,
) *WalletUseCase {
	return &WalletUseCase{
		Log:               log,
		Config:            config,
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

func (uc *WalletUseCase) DebetWallet(ctx context.Context, req *model.NotificationUser) error {
	uc.Log.Info(
		"wallet-usecase",
		fmt.Sprintf("Processing debit wallet after trip completed: %+v", req),
		"DebetWallet",
		"",
	)
	if req.EventType != "ORDER_COMPLETED" {
		uc.Log.Info("wallet-usecase", "Skip DebetWallet, eventType not ORDER_COMPLETED", "DebetWallet", req.EventType)
		return nil
	}
	order, err := uc.OrderRepository.FindOneOrder(ctx, entity.OrderFilter{
		OrderID:     &req.OrderID,
		PassengerID: &req.PassengerID,
		DriverID:    &req.DriverID,
	})
	if err != nil || order == nil {
		uc.Log.Error("wallet-usecase", "Order not found for debit", "DebetWallet", utils.ConvertString(err))
		return fmt.Errorf("order not found")
	}

	if order.PaymentMethod != "WALLET" && order.PaymentMethod != "EWALLET" {
		uc.Log.Info("wallet-usecase", "Payment method is not wallet, skip debit", "DebetWallet", order.PaymentMethod)
		return nil
	}
	if order.PaymentStatus == "PAID" {
		uc.Log.Info("wallet-usecase", "Order already paid, skip debit", "DebetWallet", order.OrderID)
		return nil
	}

	var actualPrice float64
	if order.EstimatedFare != nil && *order.EstimatedFare > 0 {
		actualPrice = *order.EstimatedFare
	} else if order.BestRoutePrice > 0 {
		actualPrice = order.BestRoutePrice
	} else {
		actualPrice = order.MaxPrice
	}

	if actualPrice <= 0 {
		uc.Log.Error("wallet-usecase", "Invalid actual price", "DebetWallet", utils.ConvertString(order))
		return fmt.Errorf("invalid actual price")
	}

	maxPrice := order.MaxPrice
	if maxPrice <= 0 {
		uc.Log.Error("wallet-usecase", "Invalid max price", "DebetWallet", utils.ConvertString(order))
		return fmt.Errorf("invalid max price")
	}

	actualPaid := actualPrice
	if actualPaid > maxPrice {
		actualPaid = maxPrice
	}

	var refundAmount float64
	if actualPrice < maxPrice {
		refundAmount = maxPrice - actualPrice
	}

	db, err := uc.DB.GetDB()
	if err != nil {
		uc.Log.Error("wallet-usecase", "failed to get db connection", "DebetWallet", utils.ConvertString(err))
		return fmt.Errorf("failed to get db connection: %v", err)
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		uc.Log.Error("wallet-usecase", "failed to start transaction", "DebetWallet", utils.ConvertString(err))
		return fmt.Errorf("failed to start transaction: %v", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	paymentTx, err := uc.PaymentRepository.FindPendingPaymentByOrder(ctx, tx.Tx, order.ID)
	if err != nil {
		_ = tx.Rollback()
		uc.Log.Error("wallet-usecase", "failed to get payment transaction", "DebetWallet", utils.ConvertString(err))
		return fmt.Errorf("failed to get payment transaction: %v", err)
	}
	if paymentTx == nil {
		_ = tx.Rollback()
		uc.Log.Error("wallet-usecase", "No pending payment transaction for order", "DebetWallet", order.OrderID)
		return fmt.Errorf("no pending payment transaction for order")
	}

	now := time.Now()

	if refundAmount > 0 {
		passengerWallet, err := uc.WalletRepository.GetWalletForUpdate(ctx, tx.Tx, req.PassengerID)
		if err != nil {
			_ = tx.Rollback()
			uc.Log.Error("wallet-usecase", "failed to get passenger wallet", "DebetWallet", utils.ConvertString(err))
			return fmt.Errorf("failed to get passenger wallet: %v", err)
		}
		if passengerWallet == nil {
			_ = tx.Rollback()
			uc.Log.Error("wallet-usecase", "Passenger wallet not found for refund", "DebetWallet", req.PassengerID)
			return fmt.Errorf("passenger wallet not found")
		}

		newPassengerBalance := passengerWallet.Balance + refundAmount
		if err := uc.WalletRepository.UpdateWalletBalance(ctx, tx.Tx, passengerWallet.ID, newPassengerBalance); err != nil {
			_ = tx.Rollback()
			uc.Log.Error("wallet-usecase", "failed to update passenger wallet balance", "DebetWallet", utils.ConvertString(err))
			return fmt.Errorf("failed to update passenger wallet balance: %v", err)
		}

		refundTrx := &entity.WalletTransaction{
			WalletID:      passengerWallet.ID,
			TransactionID: utils.GenerateUniqueIDWithPrefix("wtrx"),
			Amount:        refundAmount,
			Type:          "credit",
			Description:   fmt.Sprintf("Refund difference for order %s", req.OrderID),
			Timestamp:     now,
		}
		if err := uc.WalletRepository.InsertWalletTransaction(ctx, tx.Tx, refundTrx); err != nil {
			_ = tx.Rollback()
			uc.Log.Error("wallet-usecase", "failed to insert refund transaction", "DebetWallet", utils.ConvertString(err))
			return fmt.Errorf("failed to insert refund transaction: %v", err)
		}

		// event log refund
		refundEvent := &entity.PaymentEventLog{
			PaymentTransactionID: paymentTx.ID,
			EventType:            "REFUND",
			EventDescription:     fmt.Sprintf("Refunded %.2f to passenger for order %s", refundAmount, req.OrderID),
			RawPayload:           nil,
		}
		if err := uc.PaymentRepository.InsertPaymentEventLogTx(ctx, tx.Tx, refundEvent); err != nil {
			_ = tx.Rollback()
			uc.Log.Error("wallet-usecase", "failed to insert refund event log", "DebetWallet", utils.ConvertString(err))
			return fmt.Errorf("failed to insert refund event log: %v", err)
		}
	}

	driverWallet, err := uc.WalletRepository.GetWalletForUpdate(ctx, tx.Tx, req.DriverID)
	if err != nil {
		_ = tx.Rollback()
		uc.Log.Error("wallet-usecase", "failed to get driver wallet", "DebetWallet", utils.ConvertString(err))
		return fmt.Errorf("failed to get driver wallet: %v", err)
	}
	if driverWallet == nil {
		walletID := utils.GenerateUniqueIDWithPrefix("wlt")
		driverWallet = &entity.Wallet{
			ID:      walletID,
			UserID:  req.DriverID,
			Balance: 0,
		}
		if err := uc.WalletRepository.InsertWallet(ctx, tx.Tx, driverWallet); err != nil {
			_ = tx.Rollback()
			uc.Log.Error("wallet-usecase", "failed to create driver wallet", "DebetWallet", utils.ConvertString(err))
			return fmt.Errorf("failed to create driver wallet: %v", err)
		}
	}

	platformFeeRate := uc.Config.GetFloat64("platform.fee")
	taxRate := uc.Config.GetFloat64("platform.tax")
	platformFee := actualPaid * platformFeeRate
	taxAmount := (actualPaid - platformFee) * taxRate

	driverSettlement := actualPaid - platformFee - taxAmount
	if driverSettlement < 0 {
		driverSettlement = 0
	}
	uc.Log.Info("wallet-usecase",
		fmt.Sprintf("PlatformFeeRate=%.2f, TaxRate=%.2f, PlatformFee=%.2f, TaxAmount=%.2f, DriverSettlement=%.2f",
			platformFeeRate, taxRate, platformFee, taxAmount, driverSettlement),
		"DebetWallet", "")
	newDriverBalance := driverWallet.Balance + driverSettlement
	if err := uc.WalletRepository.UpdateWalletBalance(ctx, tx.Tx, driverWallet.ID, newDriverBalance); err != nil {
		_ = tx.Rollback()
		uc.Log.Error("wallet-usecase", "failed to update driver wallet balance", "DebetWallet", utils.ConvertString(err))
		return fmt.Errorf("failed to update driver wallet balance: %v", err)
	}

	driverTrx := &entity.WalletTransaction{
		WalletID:      driverWallet.ID,
		TransactionID: utils.GenerateUniqueIDWithPrefix("wtrx"),
		Amount:        driverSettlement,
		Type:          "credit",
		Description:   fmt.Sprintf("Trip earning for order %s", req.OrderID),
		Timestamp:     now,
	}
	if err := uc.WalletRepository.InsertWalletTransaction(ctx, tx.Tx, driverTrx); err != nil {
		_ = tx.Rollback()
		uc.Log.Error("wallet-usecase", "failed to insert driver settlement transaction", "DebetWallet", utils.ConvertString(err))
		return fmt.Errorf("failed to insert driver settlement transaction: %v", err)
	}
	settlement := &entity.PaymentSettlement{
		PaymentTransactionID: paymentTx.ID,
		DriverID:             req.DriverID,
		SettlementAmount:     driverSettlement,
		PlatformFee:          platformFee,
		TaxAmount:            taxAmount,
		Status:               "PAID",
		SettlementMethod:     "WALLET",
		CreatedAt:            time.Now(),
	}

	if err := uc.PaymentRepository.InsertPaymentSettlementTx(ctx, tx, settlement); err != nil {
		_ = tx.Rollback()
		uc.Log.Error("wallet-usecase", "failed to insert payment settlement", "DebetWallet", err.Error())
		return err
	}

	paymentTx.Amount = actualPaid
	paymentTx.PaymentStatus = "SUCCESS"
	paymentTx.PaidAt = &now

	if err := uc.PaymentRepository.UpdatePaymentTransactionTx(ctx, tx, paymentTx); err != nil {
		_ = tx.Rollback()
		uc.Log.Error("wallet-usecase", "failed to update payment transaction", "DebetWallet", utils.ConvertString(err))
		return fmt.Errorf("failed to update payment transaction: %v", err)
	}

	successEvent := &entity.PaymentEventLog{
		PaymentTransactionID: paymentTx.ID,
		EventType:            "SUCCESS",
		EventDescription:     fmt.Sprintf("Wallet payment captured %.2f for order %s", actualPaid, req.OrderID),
		RawPayload:           nil,
	}
	if err := uc.PaymentRepository.InsertPaymentEventLogTx(ctx, tx.Tx, successEvent); err != nil {
		_ = tx.Rollback()
		uc.Log.Error("wallet-usecase", "failed to insert success event log", "DebetWallet", utils.ConvertString(err))
		return fmt.Errorf("failed to insert success event log: %v", err)
	}

	ok, err := uc.OrderRepository.MarkOrderPaidTx(ctx, tx.Tx, req.OrderID, req.PassengerID, req.DriverID)
	if err != nil {
		_ = tx.Rollback()
		uc.Log.Error("wallet-usecase", "failed to update order payment status", "DebetWallet", utils.ConvertString(err))
		return fmt.Errorf("failed to update order payment status: %v", err)
	}
	if !ok {
		_ = tx.Rollback()
		uc.Log.Error("wallet-usecase", "order payment status not updated (maybe concurrent)", "DebetWallet", req.OrderID)
		return fmt.Errorf("order payment status not updated")
	}

	if err := tx.Commit(); err != nil {
		uc.Log.Error("wallet-usecase", "failed to commit transaction", "DebetWallet", utils.ConvertString(err))
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	uc.Log.Info(
		"wallet-usecase",
		fmt.Sprintf("Debit wallet + settlement success. order=%s paid=%.2f refund=%.2f", req.OrderID, actualPaid, refundAmount),
		"DebetWallet",
		"",
	)

	return nil
}

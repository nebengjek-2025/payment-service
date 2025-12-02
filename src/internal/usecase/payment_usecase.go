package usecase

import (
	"context"
	"fmt"
	"payment-service/src/pkg/databases/mysql"
	httpError "payment-service/src/pkg/http-error"
	"payment-service/src/pkg/log"
	"payment-service/src/pkg/utils"
	"time"

	"payment-service/src/internal/entity"
	"payment-service/src/internal/model"
	"payment-service/src/internal/repository"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

type PaymentUseCase struct {
	Log               log.Log
	UserRepository    *repository.UserRepository
	OrderRepository   *repository.OrderRepository
	PaymentRepository *repository.PaymentRepository
	Config            *viper.Viper
	DB                mysql.DBInterface
	Redis             redis.UniversalClient
}

func NewPaymentUseCase(
	logger log.Log,
	config *viper.Viper,
	userRepository *repository.UserRepository,
	paymentRepository *repository.PaymentRepository,
	orderRepository *repository.OrderRepository,
	db mysql.DBInterface,
	redisClient redis.UniversalClient,
) *PaymentUseCase {
	return &PaymentUseCase{
		Log:               logger,
		Config:            config,
		UserRepository:    userRepository,
		PaymentRepository: paymentRepository,
		OrderRepository:   orderRepository,
		DB:                db,
		Redis:             redisClient,
	}
}

func (uc *PaymentUseCase) GenerateQrisPayment(ctx context.Context, req *model.CreateQrisPaymentRequest) utils.Result {
	var result utils.Result

	if req.OrderID == "" || req.UserID == "" {
		errObj := httpError.NewBadRequest()
		errObj.Message = "orderId and userId are required"
		result.Error = errObj
		uc.Log.Error("payment-usecase", errObj.Message, "GenerateQrisSnap", utils.ConvertString(req))
		return result
	}

	order, err := uc.OrderRepository.FindOneOrder(ctx, entity.OrderFilter{
		OrderID:     &req.OrderID,
		PassengerID: &req.UserID,
	})
	if err != nil || order == nil {
		errObj := httpError.NewNotFound()
		errObj.Message = "order not found"
		result.Error = errObj
		uc.Log.Error("payment-usecase", errObj.Message, "GenerateQrisSnap", utils.ConvertString(err))
		return result
	}

	user, err := uc.UserRepository.FindByID(ctx, req.UserID)
	if err != nil || order == nil {
		errObj := httpError.NewNotFound()
		errObj.Message = "user not found"
		result.Error = errObj
		uc.Log.Error("payment-usecase", errObj.Message, "GenerateQrisSnap", utils.ConvertString(err))
		return result
	}

	amount := uc.calculateFinalAmount(order)
	if amount <= 0 {
		errObj := httpError.NewBadRequest()
		errObj.Message = "invalid order amount"
		result.Error = errObj
		uc.Log.Error("payment-usecase", errObj.Message, "GenerateQrisSnap", utils.ConvertString(order))
		return result
	}

	serverKey := uc.Config.GetString("midtrans.server_key")
	if serverKey == "" {
		errObj := httpError.NewInternalServerError()
		errObj.Message = "midtrans server key not configured"
		result.Error = errObj
		uc.Log.Error("payment-usecase", errObj.Message, "GenerateQrisSnap", "")
		return result
	}
	isProd := uc.Config.GetBool("midtrans.is_production")

	env := midtrans.Sandbox
	if isProd {
		env = midtrans.Production
	}

	snapClient := snap.Client{}
	snapClient.New(serverKey, env)

	snapReq := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  order.OrderID,
			GrossAmt: int64(amount),
		},
		CustomerDetail: &midtrans.CustomerDetails{
			Email: user.Email,
			FName: user.FullName,
		},
	}

	snapResp, err := snapClient.CreateTransaction(snapReq)
	if snapResp == nil {
		errObj := httpError.NewInternalServerError()
		errObj.Message = fmt.Sprintf("failed create qris via midtrans snap: %v", err)
		result.Error = errObj
		uc.Log.Error("payment-usecase", errObj.Message, "GenerateQrisSnap", utils.ConvertString(err))
		return result
	}

	snapToken := snapResp.Token
	redirectURL := snapResp.RedirectURL

	db, err := uc.DB.GetDB()
	if err != nil {
		errObj := httpError.NewInternalServerError()
		errObj.Message = "failed to get db connection"
		result.Error = errObj
		uc.Log.Error("payment-usecase", errObj.Message, "GenerateQrisSnap", utils.ConvertString(err))
		return result
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		errObj := httpError.NewInternalServerError()
		errObj.Message = "failed to start transaction"
		result.Error = errObj
		uc.Log.Error("payment-usecase", errObj.Message, "GenerateQrisSnap", utils.ConvertString(err))
		return result
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	providerName := "MIDTRANS_SNAP"
	driverID := ""
	if order.DriverID != nil {
		driverID = *order.DriverID
	}
	payment := &entity.PaymentTransaction{
		RideOrderID:   order.ID,
		PassengerID:   order.PassengerID,
		DriverID:      driverID,
		Amount:        amount,
		Currency:      "IDR",
		PaymentMethod: "QRIS",
		PaymentStatus: "PENDING",
		ProviderName:  &providerName,
	}

	paymentID, err := uc.PaymentRepository.InsertPaymentTransactionTx(ctx, tx, payment)
	if err != nil {
		_ = tx.Rollback()
		errObj := httpError.NewInternalServerError()
		errObj.Message = "failed to save payment transaction"
		result.Error = errObj
		uc.Log.Error("payment-usecase", errObj.Message, "GenerateQrisSnap", utils.ConvertString(err))
		return result
	}

	rawPayload := utils.ConvertString(snapResp)
	event := &entity.PaymentEventLog{
		PaymentTransactionID: paymentID,
		EventType:            "CREATE",
		EventDescription:     "Create QRIS Snap payment via Midtrans",
		RawPayload:           &rawPayload,
	}
	if err := uc.PaymentRepository.InsertPaymentEventLogTx(ctx, tx.Tx, event); err != nil {
		_ = tx.Rollback()
		errObj := httpError.NewInternalServerError()
		errObj.Message = "failed to save payment event log"
		result.Error = errObj
		uc.Log.Error("payment-usecase", errObj.Message, "GenerateQrisSnap", utils.ConvertString(err))
		return result
	}

	if err := tx.Commit(); err != nil {
		errObj := httpError.NewInternalServerError()
		errObj.Message = "failed to commit transaction"
		result.Error = errObj
		uc.Log.Error("payment-usecase", errObj.Message, "GenerateQrisSnap", utils.ConvertString(err))
		return result
	}

	result.Data = model.QrisSnapPaymentResponse{
		OrderID:     order.OrderID,
		Amount:      amount,
		SnapToken:   snapToken,
		RedirectURL: redirectURL,
		Status:      "PENDING",
	}

	return result
}

func (uc *PaymentUseCase) CallbackPayment(ctx context.Context, notif *model.MidtransNotification) utils.Result {
	var result utils.Result

	uc.Log.Info("payment-usecase",
		fmt.Sprintf("Received Midtrans webhook: %+v", notif),
		"HandleMidtransWebhook",
		"",
	)

	serverKey := uc.Config.GetString("midtrans.server_key")
	if serverKey == "" {
		errObj := httpError.NewInternalServerError()
		errObj.Message = "midtrans server key not configured"
		result.Error = errObj
		return result
	}

	expectedSig := utils.GenerateMidtransSignature(
		notif.OrderID,
		notif.StatusCode,
		notif.GrossAmount,
		serverKey,
	)

	if notif.SignatureKey != expectedSig {
		errObj := httpError.NewUnauthorized()
		errObj.Message = "invalid signature"
		result.Error = errObj
		uc.Log.Error("payment-usecase", errObj.Message, "HandleMidtransWebhook",
			fmt.Sprintf("expected=%s got=%s", expectedSig, notif.SignatureKey))
		return result
	}

	db, err := uc.DB.GetDB()
	if err != nil {
		errObj := httpError.NewInternalServerError()
		errObj.Message = "failed to get db connection"
		result.Error = errObj
		uc.Log.Error("payment-usecase", errObj.Message, "HandleMidtransWebhook", utils.ConvertString(err))
		return result
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		errObj := httpError.NewInternalServerError()
		errObj.Message = "failed to start transaction"
		result.Error = errObj
		uc.Log.Error("payment-usecase", errObj.Message, "HandleMidtransWebhook", utils.ConvertString(err))
		return result
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	paymentTx, err := uc.PaymentRepository.FindByOrderIDForUpdate(ctx, tx, notif.OrderID)
	if err != nil {
		_ = tx.Rollback()
		errObj := httpError.NewInternalServerError()
		errObj.Message = "failed to get payment transaction"
		result.Error = errObj
		uc.Log.Error("payment-usecase", errObj.Message, "HandleMidtransWebhook", utils.ConvertString(err))
		return result
	}
	if paymentTx == nil {
		_ = tx.Rollback()
		errObj := httpError.NewNotFound()
		errObj.Message = "payment transaction not found"
		result.Error = errObj
		uc.Log.Error("payment-usecase", errObj.Message, "HandleMidtransWebhook", notif.OrderID)
		return result
	}

	var newStatus string
	switch notif.TransactionStatus {
	case "capture", "settlement":
		if notif.FraudStatus == "challenge" {
			newStatus = "PENDING"
		} else {
			newStatus = "SUCCESS"
		}
	case "pending":
		newStatus = "PENDING"
	case "deny", "cancel", "expire":
		newStatus = "FAILED"
	case "refund", "partial_refund":
		newStatus = "REFUNDED"
	default:
		newStatus = "PENDING"
	}

	if paymentTx.PaymentStatus == newStatus {
		result.Data = map[string]string{"message": "status unchanged"}
		_ = tx.Commit()
		return result
	}

	now := time.Now()
	paymentTx.PaymentStatus = newStatus

	if newStatus == "SUCCESS" {
		paymentTx.PaidAt = &now
	}

	if err := uc.PaymentRepository.UpdatePaymentTransactionTx(ctx, tx, paymentTx); err != nil {
		_ = tx.Rollback()
		errObj := httpError.NewInternalServerError()
		errObj.Message = "failed to update payment transaction"
		result.Error = errObj
		uc.Log.Error("payment-usecase", errObj.Message, "HandleMidtransWebhook", utils.ConvertString(err))
		return result
	}

	rawPayload := utils.ConvertString(notif)
	event := &entity.PaymentEventLog{
		PaymentTransactionID: paymentTx.ID,
		EventType:            mapMidtransEventType(notif.TransactionStatus),
		EventDescription:     fmt.Sprintf("Midtrans notif: %s", notif.TransactionStatus),
		RawPayload:           &rawPayload,
	}
	if err := uc.PaymentRepository.InsertPaymentEventLogTx(ctx, tx.Tx, event); err != nil {
		_ = tx.Rollback()
		errObj := httpError.NewInternalServerError()
		errObj.Message = "failed to insert payment event log"
		result.Error = errObj
		uc.Log.Error("payment-usecase", errObj.Message, "HandleMidtransWebhook", utils.ConvertString(err))
		return result
	}

	if newStatus == "SUCCESS" {
		order, err := uc.OrderRepository.FindOneOrder(ctx, entity.OrderFilter{ID: &paymentTx.RideOrderID})
		if err != nil || order == nil {
			_ = tx.Rollback()
			errObj := httpError.NewInternalServerError()
			errObj.Message = "failed to get order for payment"
			result.Error = errObj
			uc.Log.Error("payment-usecase", errObj.Message, "HandleMidtransWebhook", utils.ConvertString(err))
			return result
		}

		if order.PaymentMethod != "WALLET" && order.PaymentMethod != "EWALLET" {
			ok, err := uc.OrderRepository.MarkOrderPaidTx(ctx, tx.Tx, order.OrderID, order.PassengerID, *order.DriverID)
			if err != nil {
				_ = tx.Rollback()
				errObj := httpError.NewInternalServerError()
				errObj.Message = "failed to update order payment status"
				result.Error = errObj
				uc.Log.Error("payment-usecase", errObj.Message, "HandleMidtransWebhook", utils.ConvertString(err))
				return result
			}
			if !ok {
				_ = tx.Rollback()
				errObj := httpError.NewConflict()
				errObj.Message = "order payment status not updated (maybe already paid/invalid state)"
				result.Error = errObj
				uc.Log.Error("payment-usecase", errObj.Message, "HandleMidtransWebhook", order.OrderID)
				return result
			}
		}
	}

	if err := tx.Commit(); err != nil {
		errObj := httpError.NewInternalServerError()
		errObj.Message = "failed to commit transaction"
		result.Error = errObj
		uc.Log.Error("payment-usecase", errObj.Message, "HandleMidtransWebhook", utils.ConvertString(err))
		return result
	}

	result.Data = map[string]string{
		"message":          "webhook processed",
		"transaction_id":   notif.TransactionID,
		"payment_status":   newStatus,
		"transaction_time": notif.TransactionTime,
	}
	return result
}

func mapMidtransEventType(status string) string {
	switch status {
	case "capture":
		return "CALLBACK"
	case "settlement":
		return "SUCCESS"
	case "pending":
		return "PENDING"
	case "deny", "cancel", "expire":
		return "FAILED"
	case "refund", "partial_refund":
		return "REFUND"
	default:
		return "CALLBACK"
	}
}

func (uc *PaymentUseCase) calculateFinalAmount(order *entity.Order) float64 {
	maxPrice := order.MaxPrice

	actualPrice := maxPrice
	if order.EstimatedFare != nil && *order.EstimatedFare > 0 {
		actualPrice = *order.EstimatedFare
	} else if order.BestRoutePrice > 0 {
		actualPrice = order.BestRoutePrice
	}

	if actualPrice > maxPrice {
		return maxPrice
	}
	if actualPrice <= 0 {
		return maxPrice
	}
	return actualPrice
}

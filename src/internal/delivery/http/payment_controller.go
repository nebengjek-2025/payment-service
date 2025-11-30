package http

import (
	"payment-service/src/internal/delivery/http/middleware"
	"payment-service/src/internal/model"
	"payment-service/src/internal/usecase"
	"payment-service/src/pkg/log"
	"payment-service/src/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type PaymentController struct {
	Log     log.Log
	UseCase *usecase.PaymentUseCase
}

func NewPaymentController(useCase *usecase.PaymentUseCase, logger log.Log) *PaymentController {
	return &PaymentController{
		Log:     logger,
		UseCase: useCase,
	}
}

func (c *PaymentController) GeneratePayment(ctx *fiber.Ctx) error {
	auth := middleware.GetUser(ctx)
	request := new(model.CreateQrisPaymentRequest)
	request.UserID = auth.UserID
	if err := ctx.BodyParser(request); err != nil {
		c.Log.Error("WalletController.TopUpWallet", "Failed to parse request body", "error", err.Error())
		return utils.ResponseError(err, ctx)
	}
	result := c.UseCase.GenerateQrisPayment(ctx.Context(), request)
	if result.Error != nil {
		return utils.ResponseError(result.Error, ctx)
	}

	return utils.Response(result.Data, "Top Up Wallet", fiber.StatusOK, ctx)
}

func (c *PaymentController) CallbackPayment(ctx *fiber.Ctx) error {
	request := new(model.MidtransNotification)
	if err := ctx.BodyParser(request); err != nil {
		c.Log.Error("WalletController.TopUpWallet", "Failed to parse request body", "error", err.Error())
		return utils.ResponseError(err, ctx)
	}
	result := c.UseCase.CallbackPayment(ctx.Context(), request)
	if result.Error != nil {
		return utils.ResponseError(result.Error, ctx)
	}

	return utils.Response(result.Data, "Top Up Wallet", fiber.StatusOK, ctx)
}

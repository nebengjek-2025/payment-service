package http

import (
	"payment-service/src/internal/delivery/http/middleware"
	"payment-service/src/internal/model"
	"payment-service/src/internal/usecase"
	"payment-service/src/pkg/log"
	"payment-service/src/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type WalletController struct {
	Log     log.Log
	UseCase *usecase.WalletUseCase
}

func NewWalletController(useCase *usecase.WalletUseCase, logger log.Log) *WalletController {
	return &WalletController{
		Log:     logger,
		UseCase: useCase,
	}
}

func (c *WalletController) TopUpWallet(ctx *fiber.Ctx) error {
	auth := middleware.GetUser(ctx)
	request := new(model.WalletRequest)
	request.UserID = auth.UserID
	if err := ctx.BodyParser(request); err != nil {
		c.Log.Error("WalletController.TopUpWallet", "Failed to parse request body", "error", err.Error())
		return utils.ResponseError(err, ctx)
	}
	result := c.UseCase.TopUpWallet(ctx.Context(), request)
	if result.Error != nil {
		return utils.ResponseError(result.Error, ctx)
	}

	return utils.Response(result.Data, "Top Up Wallet", fiber.StatusOK, ctx)
}

func (c *WalletController) GetWallet(ctx *fiber.Ctx) error {
	auth := middleware.GetUser(ctx)
	request := &model.GetUserRequest{
		ID: auth.UserID,
	}
	result := c.UseCase.GetWallet(ctx.Context(), request)
	if result.Error != nil {
		return utils.ResponseError(result.Error, ctx)
	}

	return utils.Response(result.Data, "Saldo Wallet", fiber.StatusOK, ctx)
}

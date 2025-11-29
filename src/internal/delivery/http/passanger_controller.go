package http

import (
	"payment-service/src/internal/delivery/http/middleware"
	"payment-service/src/internal/model"
	"payment-service/src/internal/usecase"
	"payment-service/src/pkg/log"
	"payment-service/src/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type PassangerController struct {
	Log     log.Log
	UseCase *usecase.PassengerUseCase
}

func NewPassangerController(useCase *usecase.PassengerUseCase, logger log.Log) *PassangerController {
	return &PassangerController{
		Log:     logger,
		UseCase: useCase,
	}
}

func (c *PassangerController) GetInboxNotification(ctx *fiber.Ctx) error {
	auth := middleware.GetUser(ctx)

	request := &model.GetUserRequest{
		ID: auth.UserID,
	}
	result := c.UseCase.GetInboxNotification(ctx.Context(), request)
	if result.Error != nil {
		return utils.ResponseError(result.Error, ctx)
	}

	return utils.Response(result.Data, "GetInboxNotification", fiber.StatusOK, ctx)
}

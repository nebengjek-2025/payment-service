package route

import (
	"payment-service/src/internal/delivery/http"
	"payment-service/src/internal/delivery/http/middleware"

	"github.com/gofiber/fiber/v2"
)

type RouteConfig struct {
	App               *fiber.App
	WalletController  *http.WalletController
	PaymentController *http.PaymentController
	AuthMiddleware    fiber.Handler
}

func (c *RouteConfig) Setup() {
	c.App.Use(middleware.NewLogger())
	c.App.Get("/health", func(ctx *fiber.Ctx) error {
		return ctx.SendString("OK")
	})
	c.SetupGuestRoute()
	c.SetupAuthRoute()
}
func (c *RouteConfig) SetupGuestRoute() {
	c.App.Post("/payment/v1/webhook", c.PaymentController.CallbackPayment)
}

func (c *RouteConfig) SetupAuthRoute() {
	c.App.Use(c.AuthMiddleware)
	c.App.Post("/wallet/v1/top-up", c.WalletController.TopUpWallet)
	c.App.Get("/wallet/v1/info", c.WalletController.GetWallet)

	c.App.Post("/order/v1/payment", c.PaymentController.GeneratePayment)
}

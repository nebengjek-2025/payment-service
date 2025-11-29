package route

import (
	"notification-service/src/internal/delivery/http"
	"notification-service/src/internal/delivery/http/middleware"

	"github.com/gofiber/fiber/v2"
)

type RouteConfig struct {
	App                 *fiber.App
	PassangerController *http.PassangerController
	AuthMiddleware      fiber.Handler
}

func (c *RouteConfig) Setup() {
	c.App.Use(middleware.NewLogger())
	c.App.Get("/health", func(ctx *fiber.Ctx) error {
		return ctx.SendString("OK")
	})
	c.SetupAuthRoute()

}

func (c *RouteConfig) SetupAuthRoute() {
	c.App.Use(c.AuthMiddleware)
	c.App.Get("/users/v1/inbox", c.PassangerController.GetInboxNotification)
}

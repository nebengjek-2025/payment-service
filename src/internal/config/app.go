package config

import (
	"payment-service/src/internal/delivery/http"
	"payment-service/src/internal/delivery/http/middleware"
	"payment-service/src/internal/delivery/http/route"

	"payment-service/src/internal/repository"
	"payment-service/src/internal/usecase"
	"payment-service/src/pkg/databases/mysql"
	kafkaPkgConfluent "payment-service/src/pkg/kafka/confluent"
	"payment-service/src/pkg/log"

	"github.com/redis/go-redis/v9"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
)

type BootstrapConfig struct {
	DB       mysql.DBInterface
	App      *fiber.App
	Log      log.Log
	Validate *validator.Validate
	Config   *viper.Viper
	Producer kafkaPkgConfluent.Producer
	Redis    redis.UniversalClient
}

func Bootstrap(config *BootstrapConfig) {
	// setup repositories
	userRepository := repository.NewUserRepository(config.DB)
	notificationRepository := repository.NewNotificationRepository(config.DB)
	orderRepository := repository.NewOrderRepository(config.DB)

	// setup use cases
	passangerUseCase := usecase.NewPassengerUseCase(
		config.Log,
		userRepository,
		notificationRepository,
		orderRepository,
		config.Redis,
	)

	// setup controller
	passangerController := http.NewPassangerController(passangerUseCase, config.Log)

	// setup middleware
	authMiddleware := middleware.VerifyBearer(config.Config)

	routeConfig := route.RouteConfig{
		App:                 config.App,
		PassangerController: passangerController,
		AuthMiddleware:      authMiddleware,
	}
	routeConfig.Setup()
}

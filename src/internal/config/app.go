package config

import (
	"notification-service/src/internal/delivery/http"
	"notification-service/src/internal/delivery/http/middleware"
	"notification-service/src/internal/delivery/http/route"

	"notification-service/src/internal/gateway/messaging"
	"notification-service/src/internal/repository"
	"notification-service/src/internal/usecase"
	"notification-service/src/pkg/databases/mysql"
	kafkaPkgConfluent "notification-service/src/pkg/kafka/confluent"
	"notification-service/src/pkg/log"

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
	walletRepository := repository.NewWalletRepository(config.DB)
	notificationRepository := repository.NewNotificationRepository(config.DB)

	userProducer := messaging.NewUserProducer(config.Producer, config.Log)

	// setup use cases
	passangerUseCase := usecase.NewPassengerUseCase(config.Log, userRepository, walletRepository, notificationRepository, config.Redis, userProducer)

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

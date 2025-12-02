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
	orderRepository := repository.NewOrderRepository(config.DB)
	walletRepository := repository.NewWalletRepository(config.DB)
	paymentRepository := repository.NewPaymentRepository(config.DB)

	// setup use cases
	walletUseCase := usecase.NewWalletUseCase(
		config.Log,
		config.Config,
		userRepository,
		orderRepository,
		walletRepository,
		paymentRepository,
		config.DB,
		config.Redis,
	)

	paymentUseCase := usecase.NewPaymentUseCase(
		config.Log,
		config.Config,
		userRepository,
		paymentRepository,
		orderRepository,
		config.DB,
		config.Redis,
	)

	// setup controller
	walletController := http.NewWalletController(walletUseCase, config.Log)
	paymentController := http.NewPaymentController(paymentUseCase, config.Log)

	// setup middleware
	authMiddleware := middleware.VerifyBearer(config.Config)

	routeConfig := route.RouteConfig{
		App:               config.App,
		WalletController:  walletController,
		PaymentController: paymentController,
		AuthMiddleware:    authMiddleware,
	}
	routeConfig.Setup()
}

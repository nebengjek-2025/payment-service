package config

import (
	"context"
	"payment-service/src/internal/delivery/messaging"
	"payment-service/src/internal/repository"
	"payment-service/src/internal/usecase"
	"payment-service/src/pkg/databases/mysql"
	kafkaPkgConfluent "payment-service/src/pkg/kafka/confluent"
	"payment-service/src/pkg/log"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

type MessagingBootstrapConfig struct {
	Ctx      context.Context
	DB       mysql.DBInterface
	Log      log.Log
	Config   *viper.Viper
	Consumer kafkaPkgConfluent.Consumer
	Redis    redis.UniversalClient
	Producer kafkaPkgConfluent.Producer
}

func BootstrapMessaging(cfg *MessagingBootstrapConfig) {
	userRepository := repository.NewUserRepository(cfg.DB)
	orderRepository := repository.NewOrderRepository(cfg.DB)
	walletRepository := repository.NewWalletRepository(cfg.DB)
	paymentRepository := repository.NewPaymentRepository(cfg.DB)

	walletUseCase := usecase.NewWalletUseCase(
		cfg.Log,
		cfg.Config,
		userRepository,
		orderRepository,
		walletRepository,
		paymentRepository,
		cfg.DB,
		cfg.Redis,
	)

	orderHandler := messaging.NewOrderConsumerHandler(
		cfg.Log,
		walletUseCase,
	)

	walletHandler := messaging.NewOrderConsumerHandler(
		cfg.Log,
		walletUseCase,
	)

	routerConfig := messaging.RouterConsumerConfig{
		Ctx:      cfg.Ctx,
		Consumer: cfg.Consumer,
		Logger:   cfg.Log,
		Handlers: map[string]kafkaPkgConfluent.ConsumerHandler{
			cfg.Config.GetString("kafka.topic.payment"): walletHandler,
			cfg.Config.GetString("kafka.topic.order"):   orderHandler,
		},
	}

	routerConfig.Setup()
}

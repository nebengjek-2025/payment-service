package config

import (
	"context"
	"notification-service/src/internal/delivery/messaging"
	"notification-service/src/internal/repository"
	"notification-service/src/internal/usecase"
	"notification-service/src/pkg/databases/mysql"
	kafkaPkgConfluent "notification-service/src/pkg/kafka/confluent"
	"notification-service/src/pkg/log"

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
	walletRepository := repository.NewWalletRepository(cfg.DB)
	driverRepository := repository.NewDriverRepository(cfg.DB)
	orderRepository := repository.NewOrderRepository(cfg.DB)
	notificationRepository := repository.NewNotificationRepository(cfg.DB)
	driverUseCase := usecase.NewDriverUseCase(
		cfg.Log,
		userRepository,
		driverRepository,
		orderRepository,
		walletRepository,
		notificationRepository,
		cfg.Redis,
	)

	passangerUsecase := usecase.NewPassengerUseCase(
		cfg.Log,
		userRepository,
		walletRepository,
		notificationRepository,
		orderRepository,
		cfg.Redis,
	)

	driverHandler := messaging.NewDriverConsumerHandler(
		cfg.Log,
		driverUseCase,
	)

	passangerHandler := messaging.NewPassangerConsumerHandler(
		cfg.Log,
		passangerUsecase,
	)

	routerConfig := messaging.RouterConsumerConfig{
		Ctx:      cfg.Ctx,
		Consumer: cfg.Consumer,
		Logger:   cfg.Log,
		Handlers: map[string]kafkaPkgConfluent.ConsumerHandler{
			cfg.Config.GetString("kafka.topic.driver"):    driverHandler,
			cfg.Config.GetString("kafka.topic.passanger"): passangerHandler,
		},
	}

	routerConfig.Setup()
}

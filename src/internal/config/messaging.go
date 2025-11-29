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
	driverRepository := repository.NewDriverRepository(cfg.DB)
	orderRepository := repository.NewOrderRepository(cfg.DB)
	notificationRepository := repository.NewNotificationRepository(cfg.DB)
	driverUseCase := usecase.NewDriverUseCase(
		cfg.Log,
		userRepository,
		driverRepository,
		orderRepository,
		notificationRepository,
		cfg.Redis,
	)

	passangerUsecase := usecase.NewPassengerUseCase(
		cfg.Log,
		userRepository,
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

	orderHandler := messaging.NewOrderConsumerHandler(
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
			cfg.Config.GetString("kafka.topic.order"):     orderHandler,
		},
	}

	routerConfig.Setup()
}

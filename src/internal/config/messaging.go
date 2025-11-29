package config

import (
	"context"
	"notification-service/src/internal/delivery/messaging"
	producer "notification-service/src/internal/gateway/messaging"
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
	notificationRepository := repository.NewNotificationRepository(cfg.DB)

	userProducer := producer.NewUserProducer(cfg.Producer, cfg.Log)

	passengerUseCase := usecase.NewPassengerUseCase(
		cfg.Log,
		userRepository,
		walletRepository,
		notificationRepository,
		cfg.Redis,
		userProducer,
	)

	passengerHandler := messaging.NewPassengerConsumerHandler(
		cfg.Log,
		passengerUseCase,
	)

	// paymentHandler := messaging.NewPaymentConsumerHandler(
	// 	cfg.Log,
	// 	paymentUseCase,
	// )

	routerConfig := messaging.RouterConsumerConfig{
		Ctx:      cfg.Ctx,
		Consumer: cfg.Consumer,
		Logger:   cfg.Log,
		Handlers: map[string]kafkaPkgConfluent.ConsumerHandler{
			cfg.Config.GetString("kafka.passenger.topic"): passengerHandler,
			// cfg.Config.GetString("kafka.payment.topic"):   paymentHandler,
		},
	}

	routerConfig.Setup()
}

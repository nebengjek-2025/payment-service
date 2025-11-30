package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"payment-service/src/internal/config"
	"payment-service/src/internal/delivery/http/middleware"
	"payment-service/src/pkg/log"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		panic(fmt.Sprintf("failed to load timezone: %v", err))
	}
	time.Local = loc
	viperConfig := config.NewViper()
	viperConfig.SetDefault("log.level", "DEBUG")
	viperConfig.SetDefault("app.name", "NOTIFICATION_SERVICE")
	viperConfig.SetDefault("web.port", 8080)

	log.InitLogger(viperConfig)
	logger := log.GetLogger()

	config.NewKafkaConfig(viperConfig)
	config.LoadRedisConfig(viperConfig)
	db := config.NewDatabase(viperConfig, logger)
	redisClient := config.NewRedis()

	producer := config.NewKafkaProducer(viperConfig, logger)
	consumer := config.NewKafkaConsumer(viperConfig, logger)

	validate := config.NewValidator(viperConfig)

	app := config.NewFiber(viperConfig)
	app.Use(middleware.NewLogger())

	config.Bootstrap(&config.BootstrapConfig{
		DB:       db,
		App:      app,
		Log:      logger,
		Validate: validate,
		Config:   viperConfig,
		Producer: producer,
		Redis:    redisClient,
	})

	config.BootstrapMessaging(&config.MessagingBootstrapConfig{
		Ctx:      ctx,
		DB:       db,
		Log:      logger,
		Config:   viperConfig,
		Consumer: consumer,
		Redis:    redisClient,
		Producer: producer,
	})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	webPort := viperConfig.GetInt("web.port")
	go func() {
		if err := app.Listen(fmt.Sprintf(":%d", webPort)); err != nil {
			logger.Error("main", fmt.Sprintf("Failed to start server: %v", err), "main", "")
		}
	}()

	<-quit
	logger.Info("main", "Server payment-service is shutting down...", "graceful", "")

	cancel()

	if err := app.Shutdown(); err != nil {
		logger.Error("main", fmt.Sprintf("Error during shutdown: %v", err), "graceful", "")
	}

	if closer, ok := consumer.(interface{ Close() error }); ok {
		_ = closer.Close()
	}

	logger.Info("main", fmt.Sprintf("Server %s stopped", viperConfig.GetString("app.name")), "gracefull", "")
}

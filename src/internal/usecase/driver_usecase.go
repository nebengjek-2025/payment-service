package usecase

import (
	"payment-service/src/pkg/log"

	"payment-service/src/internal/repository"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

type DriverUseCase struct {
	Log              log.Log
	UserRepository   *repository.UserRepository
	OrderRepository  *repository.OrderRepository
	DriverRepository *repository.DriverRepository
	Config           *viper.Viper
	Redis            redis.UniversalClient
}

func NewDriverUseCase(
	logger log.Log,
	userRepository *repository.UserRepository,
	driverRepository *repository.DriverRepository,
	orderRepository *repository.OrderRepository,
	redisClient redis.UniversalClient,
) *DriverUseCase {
	return &DriverUseCase{
		Log:              logger,
		UserRepository:   userRepository,
		DriverRepository: driverRepository,
		OrderRepository:  orderRepository,
		Redis:            redisClient,
	}
}

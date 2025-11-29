package repository

import (
	"context"
	"notification-service/src/internal/entity"
)

type Repository interface {
	FindByID(ctx context.Context, id string) (*entity.User, error)
	FindByToken(ctx context.Context, token string) (*entity.User, error)
}

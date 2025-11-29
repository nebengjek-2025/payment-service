package repository

import (
	"context"
	"payment-service/src/internal/entity"
)

type Repository interface {
	FindByID(ctx context.Context, id string) (*entity.User, error)
	FindByToken(ctx context.Context, token string) (*entity.User, error)
}

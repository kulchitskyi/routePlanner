package repository

import (
	"context"
	"routePlanner/internal/models"
)

type UserRepository interface {
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Create(ctx context.Context, u *models.User) error
	Update(ctx context.Context, id string, u *models.UserUpdateRequest) error
	Delete(ctx context.Context, id string) error
}

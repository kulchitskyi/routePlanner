package repository

import (
	"context"
	"routePlanner/internal/models"
)

type PlaceRepository interface {
	GetByID(ctx context.Context, id string) (*models.Place, error)

	GetByTags(ctx context.Context, tags []string) ([]models.Place, error)
	GetAllTags(ctx context.Context) ([]string, error)

	FindOptimalPlaces(ctx context.Context, tags []string, userLocation models.Location, radius float64) ([]models.Place, error)

	Create(ctx context.Context, p *models.Place) error
	Update(ctx context.Context, id string, p *models.PlaceUpdateRequest) error
	Delete(ctx context.Context, id string) error
}

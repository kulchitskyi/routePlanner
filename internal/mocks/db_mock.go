package mocks

import (
	"context"

	"routePlanner/internal/models"

	"github.com/stretchr/testify/mock"
)

type MockPlaceRepository struct {
	mock.Mock
}

func (m *MockPlaceRepository) GetAll(ctx context.Context) ([]models.Place, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Place), args.Error(1)
}

func (m *MockPlaceRepository) GetByID(ctx context.Context, id string) (*models.Place, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Place), args.Error(1)
}

func (m *MockPlaceRepository) Create(ctx context.Context, place *models.Place) error {
	args := m.Called(ctx, place)
	return args.Error(0)
}

func (m *MockPlaceRepository) Update(ctx context.Context, id string, req *models.PlaceUpdateRequest) error {
	args := m.Called(ctx, id, req)
	return args.Error(0)
}

func (m *MockPlaceRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPlaceRepository) DeleteAll(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockPlaceRepository) GetByTags(ctx context.Context, tags []string) ([]models.Place, error) {
	args := m.Called(ctx, tags)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Place), args.Error(1)
}

func (m *MockPlaceRepository) GetAllTags(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockPlaceRepository) FindOptimalPlaces(ctx context.Context, tags []string, location models.Location, radius float64) ([]models.Place, error) {
	args := m.Called(ctx, tags, location, radius)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Place), args.Error(1)
}

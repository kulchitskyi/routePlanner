package mocks

import (
	"context"

	"routePlanner/internal/models"

	"github.com/stretchr/testify/mock"
)

type MockRouteApiClient struct {
	mock.Mock
}

func (m *MockRouteApiClient) BuildRoute(ctx context.Context, origin, destination models.Location) (string, error) {
	args := m.Called(ctx, origin, destination)
	return args.String(0), args.Error(1)
}

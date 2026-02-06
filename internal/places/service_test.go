package places

import (
	"context"
	"errors"
	"testing"

	er "routePlanner/internal/errors"
	mocks "routePlanner/internal/mocks"
	"routePlanner/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func createTestService(repo *mocks.MockPlaceRepository) *PlaceService {
	return NewPlaceService(repo)
}

func TestGetById_Success(t *testing.T) {
	mockRepo := new(mocks.MockPlaceRepository)

	service := createTestService(mockRepo)
	ctx := context.Background()
	id := "test-place-id"

	expectedPlace := &models.Place{
		ID:      id,
		Name:    "Test Place",
		Tags:    []string{"cafe", "restaurant"},
		Address: "Test Street",
	}

	mockRepo.On("GetByID", ctx, id).Return(expectedPlace, nil)

	place, err := service.GetById(ctx, id)

	assert.NoError(t, err)
	assert.Equal(t, expectedPlace, place)
	mockRepo.AssertExpectations(t)
}

func TestGetById_NotFound(t *testing.T) {
	mockRepo := new(mocks.MockPlaceRepository)

	service := createTestService(mockRepo)
	ctx := context.Background()
	id := "non-existent-id"

	mockRepo.On("GetByID", ctx, id).Return(nil, er.ErrPlaceNotFound)

	place, err := service.GetById(ctx, id)

	assert.Error(t, err)
	assert.Nil(t, place)
	assert.ErrorIs(t, err, er.ErrPlaceNotFound)
	mockRepo.AssertExpectations(t)
}

var mockUserID = "test-user-id"

func TestCreate_Success(t *testing.T) {
	mockRepo := new(mocks.MockPlaceRepository)

	service := createTestService(mockRepo)
	ctx := context.Background()

	createReq := &models.PlaceCreateRequest{
		Name:        "New Cafe",
		Description: "A cozy cafe",
		Tags:        []string{"cafe", "coffee"},
		Location: models.Location{
			Latitude:  50.4501,
			Longitude: 30.5234,
		},
		Address: "New Street 1",
		Rating:  4.5,
	}

	mockRepo.On("Create", ctx, mock.MatchedBy(func(p *models.Place) bool {
		return p.Name == createReq.Name &&
			p.Description == createReq.Description &&
			len(p.Tags) == 2
	})).Run(func(args mock.Arguments) {
		p := args.Get(1).(*models.Place)
		p.ID = "new-uuid"
	}).Return(nil)

	place, err := service.Create(ctx, createReq, mockUserID)

	assert.NoError(t, err)
	assert.NotNil(t, place)
	assert.Equal(t, createReq.Name, place.Name)
	assert.Equal(t, createReq.Description, place.Description)
	assert.NotEmpty(t, place.ID)
	mockRepo.AssertExpectations(t)
}

func TestCreate_RepositoryError(t *testing.T) {
	mockRepo := new(mocks.MockPlaceRepository)

	service := createTestService(mockRepo)
	ctx := context.Background()

	createReq := &models.PlaceCreateRequest{
		Name:    "New Cafe",
		Tags:    []string{"cafe"},
		Address: "Test St",
		Location: models.Location{
			Latitude:  50.4501,
			Longitude: 30.5234,
		},
	}

	expectedError := errors.New("database constraint violation")
	mockRepo.On("Create", ctx, mock.Anything).Return(expectedError)

	place, err := service.Create(ctx, createReq, mockUserID)

	assert.Error(t, err)
	assert.Nil(t, place)
	assert.Equal(t, expectedError, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdate_Success(t *testing.T) {
	mockRepo := new(mocks.MockPlaceRepository)

	service := createTestService(mockRepo)
	ctx := context.Background()
	id := "test-place-id"

	updateReq := &models.PlaceUpdateRequest{
		Name:        stringPtr("Updated Name"),
		Description: stringPtr("Updated Description"),
		Rating:      float64Ptr(4.8),
	}

	mockRepo.On("Update", ctx, id, mock.MatchedBy(func(req *models.PlaceUpdateRequest) bool {
		return *req.Name == "Updated Name" &&
			*req.Description == "Updated Description" &&
			*req.Rating == 4.8
	})).Return(nil)

	err := service.Update(ctx, id, updateReq)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdate_NotFound(t *testing.T) {
	mockRepo := new(mocks.MockPlaceRepository)

	service := createTestService(mockRepo)
	ctx := context.Background()
	id := "non-existent-id"

	updateReq := &models.PlaceUpdateRequest{
		Name: stringPtr("Updated Name"),
	}

	mockRepo.On("Update", ctx, id, mock.Anything).Return(er.ErrPlaceNotFound)

	err := service.Update(ctx, id, updateReq)

	assert.Error(t, err)
	assert.ErrorIs(t, err, er.ErrPlaceNotFound)
	mockRepo.AssertExpectations(t)
}

func TestDeleteById_Success(t *testing.T) {
	mockRepo := new(mocks.MockPlaceRepository)

	service := createTestService(mockRepo)
	ctx := context.Background()
	id := "test-place-id"

	mockRepo.On("Delete", ctx, id).Return(nil)

	err := service.DeleteById(ctx, id)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestDeleteById_NotFound(t *testing.T) {
	mockRepo := new(mocks.MockPlaceRepository)

	service := createTestService(mockRepo)
	ctx := context.Background()
	id := "non-existent-id"

	mockRepo.On("Delete", ctx, id).Return(er.ErrPlaceNotFound)

	err := service.DeleteById(ctx, id)

	assert.Error(t, err)
	assert.ErrorIs(t, err, er.ErrPlaceNotFound)
	mockRepo.AssertExpectations(t)
}

func TestUpdate_NilFields(t *testing.T) {
	mockRepo := new(mocks.MockPlaceRepository)

	service := createTestService(mockRepo)
	ctx := context.Background()
	id := "test-place-id"

	updateReq := &models.PlaceUpdateRequest{
		// All fields nil
	}

	mockRepo.On("Update", ctx, id, mock.MatchedBy(func(req *models.PlaceUpdateRequest) bool {
		return req.Name == nil &&
			req.Description == nil &&
			req.Rating == nil
	})).Return(nil)

	err := service.Update(ctx, id, updateReq)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func stringPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}

func BenchmarkCreate(b *testing.B) {
	mockRepo := new(mocks.MockPlaceRepository)

	service := createTestService(mockRepo)
	ctx := context.Background()

	createReq := &models.PlaceCreateRequest{
		Name:    "Benchmark Cafe",
		Tags:    []string{"cafe"},
		Address: "Benchmark St",
		Location: models.Location{
			Latitude:  50.4501,
			Longitude: 30.5234,
		},
	}

	mockRepo.On("Create", ctx, mock.Anything).Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.Create(ctx, createReq, mockUserID)
	}
}

func TestUpdate_MultipleScenarios(t *testing.T) {
	errDB := errors.New("database error")

	tests := []struct {
		name        string
		id          string
		updateReq   *models.PlaceUpdateRequest
		repoUpdErr  error
		expectedErr error
	}{
		{
			name: "Update name only",
			id:   "place-1",
			updateReq: &models.PlaceUpdateRequest{
				Name: stringPtr("New Name"),
			},
			repoUpdErr:  nil,
			expectedErr: nil,
		},
		{
			name: "Place not found",
			id:   "place-2",
			updateReq: &models.PlaceUpdateRequest{
				Name: stringPtr("New Name"),
			},
			repoUpdErr:  er.ErrPlaceNotFound,
			expectedErr: er.ErrPlaceNotFound,
		},
		{
			name: "Update fails",
			id:   "place-3",
			updateReq: &models.PlaceUpdateRequest{
				Name: stringPtr("New Name"),
			},
			repoUpdErr:  errDB,
			expectedErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.MockPlaceRepository)

			service := createTestService(mockRepo)
			ctx := context.Background()

			mockRepo.On("Update", ctx, tt.id, mock.Anything).Return(tt.repoUpdErr)

			err := service.Update(ctx, tt.id, tt.updateReq)

			if tt.expectedErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

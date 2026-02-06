package travelroutes

import (
	"os"
	"context"
	"errors"
	"testing"
	"log/slog"

	mocks "routePlanner/internal/mocks"
	"routePlanner/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	TEST_TAGS_TTL int = 1
)

func setupServiceTest() *RouteService {
	mockRepo := new(mocks.MockPlaceRepository)
	mockCache := new(mocks.MockCache)
	mockExtractor := new(mocks.MockTagExtractor)
	mockRouteApiClient := new(mocks.MockRouteApiClient)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewRouteService(mockRepo, logger, mockCache, mockExtractor, mockRouteApiClient, TEST_TAGS_TTL)

	return service
}

func getServiceMocks(service *RouteService) (*mocks.MockPlaceRepository, *mocks.MockCache, *mocks.MockTagExtractor, *mocks.MockRouteApiClient) {
	return service.repo.(*mocks.MockPlaceRepository),
		service.cache.(*mocks.MockCache),
		service.extractor.(*mocks.MockTagExtractor),
		service.routeApiClient.(*mocks.MockRouteApiClient)
}

func TestGetAllowedTags_FromCache(t *testing.T) {
	service := setupServiceTest()
	mockRepo, mockCache, _, _ := getServiceMocks(service)
	ctx := context.Background()

	cachedTags := []string{"cafe", "park", "museum"}
	mockCache.On("Get", ctx, "allowed_tags").Return(cachedTags, nil)

	tags, err := service.GetAllowedTags(ctx)

	assert.NoError(t, err)
	assert.Equal(t, cachedTags, tags)
	mockCache.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "GetAllTags")
}

func TestGetAllowedTags_FromRepository(t *testing.T) {
	service := setupServiceTest()
	mockRepo, mockCache, _, _ := getServiceMocks(service)
	ctx := context.Background()

	repoTags := []string{"restaurant", "bar", "hotel"}
	mockCache.On("Get", ctx, "allowed_tags").Return(nil, errors.New("cache miss"))
	mockRepo.On("GetAllTags", ctx).Return(repoTags, nil)
	mockCache.On("Set", ctx, "allowed_tags", repoTags, mock.Anything).Return(nil)

	tags, err := service.GetAllowedTags(ctx)

	assert.NoError(t, err)
	assert.Equal(t, repoTags, tags)
	mockCache.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestGetAllowedTags_RepositoryError(t *testing.T) {
	service := setupServiceTest()
	mockRepo, mockCache, _, _ := getServiceMocks(service)
	ctx := context.Background()

	expectedError := errors.New("database error")
	mockCache.On("Get", ctx, "allowed_tags").Return(nil, errors.New("cache miss"))
	mockRepo.On("GetAllTags", ctx).Return(nil, expectedError)

	tags, err := service.GetAllowedTags(ctx)

	assert.Error(t, err)
	assert.Nil(t, tags)
	assert.Equal(t, expectedError, err)
	mockCache.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestGetByTags_Success(t *testing.T) {
	service := setupServiceTest()
	mockRepo, mockCache, mockExtractor, _ := getServiceMocks(service)
	ctx := context.Background()

	userInput := "I want to visit a cafe and then a park"
	allowedTags := []string{"cafe", "park", "museum"}
	extractedTags := []string{"cafe", "park"}

	places := []models.Place{
		{ID: "1", Name: "Test Cafe", Tags: []string{"cafe"}},
		{ID: "2", Name: "Test Park", Tags: []string{"park"}},
	}

	mockCache.On("Get", ctx, "allowed_tags").Return(allowedTags, nil)
	mockExtractor.On("ExtractTags", userInput, allowedTags).Return(extractedTags, nil)
	mockRepo.On("GetByTags", ctx, extractedTags).Return(places, nil)

	result, err := service.GetByTags(ctx, userInput)

	assert.NoError(t, err)
	assert.Equal(t, places, result)
	mockCache.AssertExpectations(t)
	mockExtractor.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestGetByTags_ExtractionError(t *testing.T) {
	service := setupServiceTest()
	_, mockCache, mockExtractor, _ := getServiceMocks(service)
	ctx := context.Background()

	userInput := "invalid input"
	allowedTags := []string{"cafe", "park"}
	expectedError := errors.New("extraction failed")

	mockCache.On("Get", ctx, "allowed_tags").Return(allowedTags, nil)
	mockExtractor.On("ExtractTags", userInput, allowedTags).Return(nil, expectedError)

	result, err := service.GetByTags(ctx, userInput)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedError, err)
	mockExtractor.AssertExpectations(t)
}

func TestAnalyzeRequest_Success(t *testing.T) {
	service := setupServiceTest()
	_, mockCache, mockExtractor, _ := getServiceMocks(service)
	ctx := context.Background()

	userInput := "cafe then park and museum"
	allowedTags := []string{"cafe", "park", "museum"}
	segments := []string{"cafe", "park", "museum"}
	extractedTags := [][]string{
		{"cafe"},
		{"park"},
		{"museum"},
	}

	mockCache.On("Get", ctx, "allowed_tags").Return(allowedTags, nil)
	mockExtractor.On("ExtractMultiTags", segments, allowedTags).Return(extractedTags, nil)

	result, err := service.AnalyzeRequest(ctx, userInput)

	assert.NoError(t, err)
	assert.Equal(t, extractedTags, result)
	mockCache.AssertExpectations(t)
	mockExtractor.AssertExpectations(t)
}

func TestAnalyzeRequest_GetAllowedTagsError(t *testing.T) {
	service := setupServiceTest()
	mockRepo, mockCache, _, _ := getServiceMocks(service)
	ctx := context.Background()

	userInput := "cafe then park"
	expectedError := errors.New("failed to get tags")

	mockCache.On("Get", ctx, "allowed_tags").Return(nil, expectedError)
	mockRepo.On("GetAllTags", ctx).Return(nil, expectedError)

	result, err := service.AnalyzeRequest(ctx, userInput)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedError, err)
}

func TestBuildRouteFromTags_Success(t *testing.T) {
	service := setupServiceTest()
	mockRepo, _, _, mockAPIClient := getServiceMocks(service)
	ctx := context.Background()

	allTags := [][]string{
		{"cafe"},
	}
	startCoords := models.Location{Latitude: 50.4501, Longitude: 30.5234}
	radius := 1000.0

	place := models.Place{
		ID:   "1",
		Name: "Test Cafe",
		Tags: []string{"cafe"},
		Location: models.Location{
			Latitude:  50.4511,
			Longitude: 30.5244,
		},
	}

	mockRepo.On("FindOptimalPlaces", ctx, []string{"cafe"}, startCoords, radius).Return([]models.Place{place}, nil)

	mockAPIClient.On("BuildRoute", ctx, startCoords, place.Location).Return(`{"type":"FeatureCollection","features":[{"type":"Feature","geometry":{"type":"LineString","coordinates":[[30.5234,50.4501],[30.5244,50.4511]]}}]}`, nil)

	result, err := service.BuildRouteFromTags(ctx, allTags, startCoords, radius)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Test Cafe")
	mockRepo.AssertExpectations(t)
	mockAPIClient.AssertExpectations(t)
}

func TestBuildRouteFromTags_API500Error(t *testing.T) {
	service := setupServiceTest()
	mockRepo, _, _, mockAPIClient := getServiceMocks(service)
	ctx := context.Background()

	allTags := [][]string{
		{"cafe"},
	}
	startCoords := models.Location{Latitude: 50.4501, Longitude: 30.5234}
	radius := 1000.0

	place := models.Place{
		ID:   "1",
		Name: "Test Cafe",
		Location: models.Location{
			Latitude:  50.4511,
			Longitude: 30.5244,
		},
	}

	mockRepo.On("FindOptimalPlaces", ctx, []string{"cafe"}, startCoords, radius).Return([]models.Place{place}, nil)

	mockAPIClient.On("BuildRoute", ctx, startCoords, place.Location).Return("", errors.New("API returned 500 Internal Server Error"))

	result, err := service.BuildRouteFromTags(ctx, allTags, startCoords, radius)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "500")
	mockRepo.AssertExpectations(t)
	mockAPIClient.AssertExpectations(t)
}

func TestBuildRouteFromTags_APIMalformedJSON(t *testing.T) {
	service := setupServiceTest()
	mockRepo, _, _, mockAPIClient := getServiceMocks(service)
	ctx := context.Background()

	allTags := [][]string{
		{"cafe"},
	}
	startCoords := models.Location{Latitude: 50.4501, Longitude: 30.5234}
	radius := 1000.0

	place := models.Place{
		ID:   "1",
		Name: "Test Cafe",
		Location: models.Location{
			Latitude:  50.4511,
			Longitude: 30.5244,
		},
	}

	mockRepo.On("FindOptimalPlaces", ctx, []string{"cafe"}, startCoords, radius).Return([]models.Place{place}, nil)

	mockAPIClient.On("BuildRoute", ctx, startCoords, place.Location).Return("{invalid json response", nil)

	result, err := service.BuildRouteFromTags(ctx, allTags, startCoords, radius)

	assert.Error(t, err)
	assert.Empty(t, result)
	mockRepo.AssertExpectations(t)
	mockAPIClient.AssertExpectations(t)
}

func TestBuildRouteFromTags_APITimeout(t *testing.T) {
	service := setupServiceTest()
	mockRepo, _, _, mockAPIClient := getServiceMocks(service)
	ctx := context.Background()

	allTags := [][]string{
		{"cafe"},
	}
	startCoords := models.Location{Latitude: 50.4501, Longitude: 30.5234}
	radius := 1000.0

	place := models.Place{
		ID:   "1",
		Name: "Test Cafe",
		Location: models.Location{
			Latitude:  50.4511,
			Longitude: 30.5244,
		},
	}

	mockRepo.On("FindOptimalPlaces", ctx, []string{"cafe"}, startCoords, radius).Return([]models.Place{place}, nil)

	mockAPIClient.On("BuildRoute", ctx, startCoords, place.Location).Return("", errors.New("context deadline exceeded"))

	result, err := service.BuildRouteFromTags(ctx, allTags, startCoords, radius)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "deadline")
	mockRepo.AssertExpectations(t)
	mockAPIClient.AssertExpectations(t)
}

func TestBuildRouteFromTags_NoPlacesFound(t *testing.T) {
	service := setupServiceTest()
	mockRepo, _, _, _ := getServiceMocks(service)
	ctx := context.Background()

	allTags := [][]string{
		{"nonexistent"},
	}
	startCoords := models.Location{Latitude: 50.4501, Longitude: 30.5234}
	radius := 1000.0

	mockRepo.On("FindOptimalPlaces", ctx, []string{"nonexistent"}, startCoords, radius).Return([]models.Place{}, nil)

	result, err := service.BuildRouteFromTags(ctx, allTags, startCoords, radius)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "no places found")
	mockRepo.AssertExpectations(t)
}

func TestBuildRouteFromTags_RepositoryError(t *testing.T) {
	service := setupServiceTest()
	mockRepo, _, _, _ := getServiceMocks(service)
	ctx := context.Background()

	allTags := [][]string{
		{"cafe"},
	}
	startCoords := models.Location{Latitude: 50.4501, Longitude: 30.5234}
	radius := 1000.0
	expectedError := errors.New("repository error")

	mockRepo.On("FindOptimalPlaces", ctx, []string{"cafe"}, startCoords, radius).Return(nil, expectedError)

	result, err := service.BuildRouteFromTags(ctx, allTags, startCoords, radius)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Equal(t, expectedError, err)
	mockRepo.AssertExpectations(t)
}

func TestSplitSegments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "English separators",
			input:    "cafe then park and museum",
			expected: []string{"cafe", "park", "museum"},
		},
		{
			name:     "Ukrainian separators",
			input:    "кафе та потім парк і музей",
			expected: []string{"кафе", "парк", "музей"},
		},
		{
			name:     "Mixed separators",
			input:    "cafe then парк and музей",
			expected: []string{"cafe", "парк", "музей"},
		},
		{
			name:     "No separators",
			input:    "cafe",
			expected: []string{"cafe"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitSegments(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func BenchmarkGetAllowedTags_Cache(b *testing.B) {
	service := setupServiceTest()
	_, mockCache, _, _ := getServiceMocks(service)
	ctx := context.Background()

	cachedTags := []string{"cafe", "park", "museum", "restaurant"}
	mockCache.On("Get", ctx, "allowed_tags").Return(cachedTags, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetAllowedTags(ctx)
	}
}

func BenchmarkSplitSegments(b *testing.B) {
	input := "cafe then park and museum after restaurant"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = splitSegments(input)
	}
}

package travelroutes

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"log/slog"
	"slices"
	"time"

	repo "routePlanner/internal/repository"
	er "routePlanner/internal/errors"
	"routePlanner/internal/models"
	
	"github.com/prometheus/client_golang/prometheus"
)

var (
	CacheHitCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "travelroutes",
			Subsystem: "cache",
			Name:      "cache_hits_total",
			Help:      "Total number of cache hits",
		},
	)
	CacheMissCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "travelroutes",
			Subsystem: "cache",
			Name:      "cache_misses_total",
			Help:      "Total number of cache misses",
		},
	)
)

func init() {
	prometheus.MustRegister(CacheHitCounter, CacheMissCounter)
}

type Cache[T any] interface {
	Get(ctx context.Context, key string) (T, error)
	Set(ctx context.Context, key string, value T, ttl time.Duration) error
}

type TagExtractor interface {
	ExtractTags(text string, allowedTags []string) ([]string, error)
	ExtractMultiTags(segments, allowedTags []string) ([][]string, error)
}

type RouterApiClient interface {
	BuildRoute(ctx context.Context, origin, destination models.Location) (string, error)
}

type RouteService struct {
	repo           repo.PlaceRepository
	logger         *slog.Logger
	extractor      TagExtractor
	cache          Cache[[]string]
	routeApiClient RouterApiClient
	tagsTTL        int
}

func NewRouteService(
	repo repo.PlaceRepository,
	logger *slog.Logger,
	cache Cache[[]string],
	extractor TagExtractor,
	routeApiClient RouterApiClient,
	tagsTTL int,
) *RouteService {
	return &RouteService{
		repo:           repo,
		logger:         logger,
		extractor:      extractor,
		cache:          cache,
		routeApiClient: routeApiClient,
		tagsTTL:        tagsTTL,
	}
}

func (s *RouteService) GetAllowedTags(ctx context.Context) ([]string, error) {
	tags, err := s.cache.Get(ctx, "allowed_tags")
	if err == nil {
		CacheHitCounter.Inc()
		return tags, nil
	}
	CacheMissCounter.Inc()

	tags, err = s.repo.GetAllTags(ctx)
	if err != nil {
		return nil, err
	}
	s.logger.Debug("Allowed tags", "tags", tags)
	err = s.cache.Set(ctx, "allowed_tags", tags, time.Duration(s.tagsTTL)*time.Minute)
	if err != nil {
		s.logger.Warn(err.Error())
	}
	return tags, nil
}

func (s *RouteService) GetByTags(ctx context.Context, userInput string) ([]models.Place, error) {

	allowedTags, err := s.GetAllowedTags(ctx)

	if err != nil {
		return nil, err
	}

	tags, err := s.extractor.ExtractTags(userInput, allowedTags)

	if err != nil {
		return nil, err
	}
	return s.repo.GetByTags(ctx, tags)
}

func (s *RouteService) AnalyzeRequest(ctx context.Context, userInput string) ([][]string, error) {
	allowedTags, err := s.GetAllowedTags(ctx)
	if err != nil {
		return nil, err
	}
	segments := splitSegments(userInput)
	allTags, err := s.extractor.ExtractMultiTags(segments, allowedTags)
	if err != nil {
		s.logger.Error("Failed to exract tags", "error", err)
		return nil, err
	}

	hasTags := false
	for _, tags := range allTags {
		if len(tags) > 0 {
			hasTags = true
			break
		}
	}

	if !hasTags {
		s.logger.Debug("failed to extract tags from preferences", "perferences", userInput)
		return nil, er.ErrExtractionFailed
	}
	return allTags, nil
}

var segmentSplitRegex = regexp.MustCompile(`(?i)(?:^|[^\p{L}])(then|and then|after|and|потім|та потім|і потім|та|і)(?:$|[^\p{L}])`)

func splitSegments(userInput string) []string {
	rawSegments := segmentSplitRegex.Split(userInput, -1)

	var segments []string
	for _, s := range rawSegments {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			segments = append(segments, trimmed)
		}
	}
	return segments
}

func (s *RouteService) BuildRouteFromTags(ctx context.Context, allTags [][]string, startCoords models.Location, radius float64) (string, error) {
	prevLoc := startCoords
	usedPlaces := make([]models.Place, 0, len(allTags))

	for _, tags := range allTags {

		places, err := s.repo.FindOptimalPlaces(ctx, tags, prevLoc, radius)

		if err != nil {
			return "", err
		}

		if len(places) == 0 {
			return "", er.ErrNoPlacesFound
		}

		var chosenPlace models.Place
		found := false
		for _, p := range places {
			isUsed := slices.ContainsFunc(usedPlaces, func(used models.Place) bool {
				return used.ID == p.ID
			})

			if !isUsed {
				chosenPlace = p
				usedPlaces = append(usedPlaces, p)
				found = true
				break
			}
		}

		if !found {
			chosenPlace = places[0]
		}

		prevLoc = chosenPlace.Location
	}

	var fullRoute []map[string]any

	prevLoc = startCoords

	for _, place := range usedPlaces {

		route, err := s.routeApiClient.BuildRoute(ctx, prevLoc, place.Location)

		if err != nil {
			return "", err
		}

		var routeSegment map[string]any

		err = json.Unmarshal([]byte(route), &routeSegment)

		if err != nil {
			return "", err
		}

		routeSegment["name"] = place.Name
		routeSegment["description"] = place.Description

		fullRoute = append(fullRoute, routeSegment)

		prevLoc = place.Location
	}

	geoJSONWithMeta, err := json.Marshal(map[string]any{
		"type":     "FeatureCollection",
		"segments": fullRoute,
	})
	return string(geoJSONWithMeta), err
}

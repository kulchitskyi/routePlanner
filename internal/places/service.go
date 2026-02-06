package places

import (
	"context"

	er "routePlanner/internal/errors"
	"routePlanner/internal/models"
	repo "routePlanner/internal/repository"
)

type PlaceService struct {
	repo repo.PlaceRepository
}

func NewPlaceService(repo repo.PlaceRepository) *PlaceService {
	return &PlaceService{repo: repo}
}

func (s *PlaceService) GetById(ctx context.Context, id string) (*models.Place, error) {
	if id == "" {
		return nil, er.ErrInvalidPlaceData
	}
	return s.repo.GetByID(ctx, id)
}

func (s *PlaceService) Create(ctx context.Context, p *models.PlaceCreateRequest, userID string) (*models.Place, error) {
	if p == nil || p.Name == "" {
		return nil, er.ErrInvalidPlaceData
	}

	place := models.Place{
		Name:        p.Name,
		Tags:        p.Tags,
		Description: p.Description,
		Location:    p.Location,
		Address:     p.Address,
		Rating:      p.Rating,
		CreatedBy:   userID,
	}

	if err := s.repo.Create(ctx, &place); err != nil {
		return nil, err
	}
	return &place, nil
}

func (s *PlaceService) Update(ctx context.Context, id string, p *models.PlaceUpdateRequest) error {
	if p == nil || id == "" {
		return er.ErrInvalidPlaceData
	}
	return s.repo.Update(ctx, id, p)
}

func (s *PlaceService) DeleteById(ctx context.Context, id string) error {
	if id == "" {
		return er.ErrInvalidPlaceData
	}
	return s.repo.Delete(ctx, id)
}

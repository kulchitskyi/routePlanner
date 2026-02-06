package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	er "routePlanner/internal/errors"
	"routePlanner/internal/models"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type PostgresPlaceRepository struct {
	DB *gorm.DB
}

func NewPostgresPlaceRepository(db *gorm.DB) *PostgresPlaceRepository {
	return &PostgresPlaceRepository{DB: db}
}

func (r *PostgresPlaceRepository) GetByID(ctx context.Context, id string) (*models.Place, error) {
	var place models.Place
	if err := r.DB.WithContext(ctx).Where("id = ?", id).First(&place).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Debug("Place not found", "id", id)
			return nil, er.ErrPlaceNotFound
		}
		slog.Error("Failed to get place by id", "id", id, "error", err)
		return nil, err
	}
	return &place, nil
}

func (r *PostgresPlaceRepository) GetByTags(ctx context.Context, tags []string) ([]models.Place, error) {
	type scoredResult struct {
		models.Place
		MatchCount int `gorm:"column:match_count"`
	}

	var res []scoredResult

	err := r.DB.WithContext(ctx).Raw(`
        SELECT p.*, (
            SELECT count(*) 
            FROM jsonb_array_elements_text(p.tags) AS t(tag) 
            WHERE t.tag = ANY($1::text[])
        ) as match_count
        FROM places p
        WHERE EXISTS (
            SELECT 1 FROM jsonb_array_elements_text(p.tags) AS t(tag)
            WHERE t.tag = ANY($1::text[])
        )
		AND p.deleted_at IS NULL
        ORDER BY match_count DESC
    `, pq.Array(tags)).Scan(&res).Error

	if err != nil {
		slog.Error("Failed to get places by tags", "error", err)
		return nil, err
	}

	places := make([]models.Place, len(res))
	for i, item := range res {
		places[i] = item.Place
	}
	return places, nil
}

func (r *PostgresPlaceRepository) GetAllTags(ctx context.Context) ([]string, error) {
	var tags []string

	err := r.DB.WithContext(ctx).Table("places").
		Select("DISTINCT jsonb_array_elements_text(tags) AS tag_name").
		Scan(&tags).Error

	slog.Debug("Allowed tags", "tags", tags)
	if err != nil {
		slog.Error("Failed to get all tags", "error", err)
		return nil, err
	}
	return tags, nil
}

func (r *PostgresPlaceRepository) FindOptimalPlaces(ctx context.Context, tags []string, userLocation models.Location, radius float64) ([]models.Place, error) {
	type scoredResult struct {
		models.Place
		MatchCount int `gorm:"column:match_count"`
	}

	var res []scoredResult

	slog.Debug("Executing query", "query", fmt.Sprintf(`
        SELECT *, (
            SELECT count(*) 
            FROM jsonb_array_elements_text(tags) AS t(tag) 
            WHERE tag = ANY(%v)
        ) as match_count
        FROM places
        WHERE tags ?| %v
		AND deleted_at IS NULL
        AND ST_DWithin(location, ST_MakePoint(%f, %f)::geography, %f)
		ORDER BY match_count DESC
    `, pq.Array(tags), pq.Array(tags),
		userLocation.Longitude, userLocation.Latitude, radius))

	err := r.DB.WithContext(ctx).Raw(`
        SELECT p.*, (
            SELECT count(*) 
            FROM jsonb_array_elements_text(p.tags) AS t(tag) 
            WHERE t.tag = ANY($1::text[])
        ) as match_count
        FROM places p
        WHERE EXISTS (
            SELECT 1 FROM jsonb_array_elements_text(p.tags) AS t(tag)
            WHERE t.tag = ANY($1::text[])
        )
		AND p.deleted_at IS NULL
        AND ST_DWithin(p.location, ST_MakePoint($2, $3)::geography, $4)
		ORDER BY match_count DESC
    `, pq.Array(tags),
		userLocation.Longitude, userLocation.Latitude, radius).Scan(&res).Error

	if err != nil {
		slog.Error("Failed to find optimal places", "error", err)
		return nil, err
	}

	places := make([]models.Place, len(res))
	for i, item := range res {
		places[i] = item.Place
	}
	return places, nil
}

func (r *PostgresPlaceRepository) Create(ctx context.Context, p *models.Place) error {
	return r.DB.WithContext(ctx).Create(p).Error
}

func (r *PostgresPlaceRepository) Update(ctx context.Context, id string, p *models.PlaceUpdateRequest) error {
	updates := map[string]interface{}{}

	if len(updates) == 0 {
		_, err := r.GetByID(ctx, id)
		return err
	}

	if p.Name != nil {
		updates["name"] = *p.Name
	}
	if p.Description != nil {
		updates["description"] = *p.Description
	}
	if p.Location != nil {
		updates["location"] = *p.Location
	}
	if p.Address != nil {
		updates["address"] = *p.Address
	}
	if p.Rating != nil {
		updates["rating"] = *p.Rating
	}

	result := r.DB.WithContext(ctx).Model(&models.Place{}).Where("id = ?", id).Updates(updates)

	if result.Error != nil {
		slog.Error("Failed to update place", "id", id, "error", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		slog.Warn("Attempted to update non-existent place", "id", id)
		return er.ErrPlaceNotFound
	}
	return nil
}

func (r *PostgresPlaceRepository) Delete(ctx context.Context, id string) error {
	result := r.DB.WithContext(ctx).Where("id = ?", id).Delete(&models.Place{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		slog.Warn("Attempted to delete non-existent place", "id", id)
		return er.ErrPlaceNotFound
	}
	return nil
}

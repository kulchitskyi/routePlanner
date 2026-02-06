package travelroutes

import (
	"errors"
	"log/slog"

	er "routePlanner/internal/errors"
	"routePlanner/internal/models"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *RouteService
	logger  *slog.Logger
}

func NewHandler(service *RouteService, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

type RoutePreferencesRequest struct {
	Preferences string `json:"preferences"`
}

type BuildRouteRequest struct {
	Tags        [][]string      `json:"tags"`
	StartCoords models.Location `json:"location"`
	Radius      float64         `json:"radius"`
}

// POST routes/analyze
func (h *Handler) AnalyzeRequest(c *fiber.Ctx) error {
	var userInput RoutePreferencesRequest

	if err := c.BodyParser(&userInput); err != nil {
		h.logger.Error("Failed to parse JSON", "error", err)
		return er.InvalidJSON(c)
	}

	tags, err := h.service.AnalyzeRequest(c.UserContext(), userInput.Preferences)

	if err != nil {
		h.logger.Error("Failed to analyze request", "error", err)
		if errors.Is(err, er.ErrExtractionFailed) {
			return er.BadRequestErr(c, err)
		}
		return er.InternalError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(tags)
}

// POST routes/build
func (h *Handler) BuildRoute(c *fiber.Ctx) error {
	var req BuildRouteRequest

	if err := c.BodyParser(&req); err != nil {
		h.logger.Error("Failed to parse JSON", "error", err)
		return er.InvalidJSON(c)
	}

	res, err := h.service.BuildRouteFromTags(c.UserContext(), req.Tags, req.StartCoords, req.Radius)

	if err != nil {
		h.logger.Error("Failed to build route", "error", err)
		if errors.Is(err, er.ErrNoPlacesFound) {
			return er.NotFound(c, err.Error())
		}
		return er.InternalError(c, err)
	}
	return c.Status(fiber.StatusOK).SendString(res)
}

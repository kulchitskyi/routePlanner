package places

import (
	"errors"
	"log/slog"

	er "routePlanner/internal/errors"
	"routePlanner/internal/models"
	"routePlanner/internal/validation"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *PlaceService
	logger  *slog.Logger
}

func NewHandler(service *PlaceService, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// GET /places/{id}
func (h *Handler) GetById(c *fiber.Ctx) error {
	id, err := validation.ValidateID(c.Params("id"))
	if err != nil {
		h.logger.Error("Failed to validate ID", "error", err)
		return er.BadRequestErr(c, err)
	}

	p, err := h.service.GetById(c.UserContext(), id)
	if err != nil {
		if errors.Is(err, er.ErrPlaceNotFound) {
			h.logger.Error("Place not found", "error", err)
			return er.NotFound(c, "Place not found")
		}
		h.logger.Error("Failed to get place by ID", "error", err)
		return er.InternalError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(p)
}

// POST /places
func (h *Handler) Create(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return er.Unauthorized(c, "Authentication required")
	}

	var p models.PlaceCreateRequest

	if err := c.BodyParser(&p); err != nil {
		h.logger.Error("Failed to parse JSON", "error", err)
		return er.InvalidJSON(c)
	}

	if errorsMap := validation.ValidateRequest(p); errorsMap != nil {
		h.logger.Error("Validation failed", "errors", errorsMap)
		return er.SendValidationErrors(c, errorsMap)
	}

	place, err := h.service.Create(c.UserContext(), &p, userID)
	if err != nil {
		h.logger.Error("Failed to create place", "error", err)
		return er.BadRequestErr(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(place)
}

// PATCH /places/{id}
func (h *Handler) Update(c *fiber.Ctx) error {
	id, err := validation.ValidateID(c.Params("id"))
	if err != nil {
		h.logger.Error("Failed to validate ID", "error", err)
		return er.BadRequestErr(c, err)
	}

	var p models.PlaceUpdateRequest
	if err = c.BodyParser(&p); err != nil {
		h.logger.Error("Failed to parse JSON", "error", err)
		return er.InvalidJSON(c)
	}

	if errorsMap := validation.ValidateRequest(p); errorsMap != nil {
		h.logger.Error("Validation failed", "errors", errorsMap)
		return er.SendValidationErrors(c, errorsMap)
	}

	err = h.service.Update(c.UserContext(), id, &p)
	if err != nil {
		if errors.Is(err, er.ErrPlaceNotFound) {
			h.logger.Error("Place not found", "error", err)
			return er.NotFound(c, "Place not found")
		}
		h.logger.Error("Failed to update place", "error", err)
		return er.InternalError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "updated"})
}

// DELETE /places/{id}
func (h *Handler) DeleteById(c *fiber.Ctx) error {
	id, err := validation.ValidateID(c.Params("id"))
	if err != nil {
		h.logger.Error("Failed to validate ID", "error", err)
		return er.BadRequestErr(c, err)
	}

	err = h.service.DeleteById(c.UserContext(), id)
	if err != nil {
		if errors.Is(err, er.ErrPlaceNotFound) {
			h.logger.Error("Place not found", "error", err)
			return er.NotFound(c, "Place not found")
		}
		h.logger.Error("Failed to delete place", "error", err)
		return er.InternalError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "deleted"})
}

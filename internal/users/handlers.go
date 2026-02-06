package users

import (
	"errors"
	"log/slog"

	er "routePlanner/internal/errors"
	"routePlanner/internal/models"
	"routePlanner/internal/validation"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *UserService
	logger  *slog.Logger
}

func NewHandler(service *UserService, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// GET /users/{id}
func (h *Handler) GetById(c *fiber.Ctx) error {
	id, err := validation.ValidateID(c.Params("id"))
	if err != nil {
		h.logger.Error("Failed to validate ID", "error", err)
		return er.BadRequestErr(c, err)
	}

	u, err := h.service.GetById(c.UserContext(), id)
	if err != nil {
		if errors.Is(err, er.ErrUserNotFound) {
			h.logger.Info("User not found", "id", id)
			return er.NotFound(c, "User not found")
		}
		h.logger.Error("Failed to get user by ID", "error", err)
		return er.InternalError(c, err)
	}

	return c.JSON(u)
}

// POST /users
func (h *Handler) Create(c *fiber.Ctx) error {
	var u models.UserCreateRequest

	if err := c.BodyParser(&u); err != nil {
		h.logger.Error("Failed to parse JSON", "error", err)
		return er.InvalidJSON(c)
	}

	if errorsMap := validation.ValidateRequest(u); errorsMap != nil {
		h.logger.Error("Validation failed", "errors", errorsMap)
		return er.SendValidationErrors(c, errorsMap)
	}

	user, err := h.service.Create(c.UserContext(), &u)
	if err != nil {
		if errors.Is(err, er.ErrInvalidUserData) || errors.Is(err, er.ErrConflict) {
			h.logger.Debug("Failed to create user", "error", err)
			return er.BadRequestErr(c, err)
		}
		h.logger.Error("Failed to create user", "error", err)
		return er.InternalError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(user)
}

// PATCH /users/{id}
func (h *Handler) Update(c *fiber.Ctx) error {
	id, err := validation.ValidateID(c.Params("id"))
	if err != nil {
		h.logger.Error("Failed to validate ID", "error", err)
		return er.BadRequestErr(c, err)
	}

	var u models.UserUpdateRequest
	if err := c.BodyParser(&u); err != nil {
		h.logger.Error("Failed to parse JSON", "error", err)
		return er.InvalidJSON(c)
	}

	if errorsMap := validation.ValidateRequest(u); errorsMap != nil {
		h.logger.Error("Validation failed", "errors", errorsMap)
		return er.SendValidationErrors(c, errorsMap)
	}

	err = h.service.Update(c.UserContext(), id, &u)
	if err != nil {
		if errors.Is(err, er.ErrUserNotFound) {
			h.logger.Info("User not found", "id", id)
			return er.NotFound(c, "User not found")
		}
		h.logger.Error("Failed to update user", "error", err)
		return er.InternalError(c, err)
	}

	return c.JSON(fiber.Map{"status": "updated"})
}

// DELETE /users/{id}
func (h *Handler) DeleteById(c *fiber.Ctx) error {
	id, err := validation.ValidateID(c.Params("id"))
	if err != nil {
		h.logger.Error("Failed to validate ID", "error", err)
		return er.BadRequestErr(c, err)
	}

	err = h.service.DeleteById(c.UserContext(), id)
	if err != nil {
		if errors.Is(err, er.ErrUserNotFound) {
			h.logger.Info("User not found", "id", id)
			return er.NotFound(c, "User not found")
		}
		h.logger.Error("Failed to delete user", "error", err)
		return er.InternalError(c, err)
	}

	return c.JSON(fiber.Map{"status": "deleted"})
}

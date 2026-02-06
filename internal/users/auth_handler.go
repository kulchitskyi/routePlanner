package users

import (
	"errors"
	"log/slog"

	"routePlanner/internal/auth"
	er "routePlanner/internal/errors"
	"routePlanner/internal/models"
	"routePlanner/internal/validation"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	service    *UserService
	jwtService *auth.JWTService
	logger     *slog.Logger
}

func NewAuthHandler(service *UserService, jwtService *auth.JWTService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		service:    service,
		jwtService: jwtService,
		logger:     logger,
	}
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req models.UserCreateRequest

	if err := c.BodyParser(&req); err != nil {
		h.logger.Error("Failed to parse JSON", "error", err)
		return er.InvalidJSON(c)
	}

	if errorsMap := validation.ValidateRequest(req); errorsMap != nil {
		h.logger.Error("Validation failed", "errors", errorsMap)
		return er.SendValidationErrors(c, errorsMap)
	}

	user, err := h.service.Create(c.UserContext(), &req)
	if err != nil {
		if errors.Is(err, er.ErrConflict) {
			return er.SendError(c, fiber.StatusConflict, err.Error())
		}
		h.logger.Error("Failed to create user", "error", err)
		return er.InternalError(c, err)
	}

	token, err := h.jwtService.GenerateToken(user.ID)
	if err != nil {
		h.logger.Error("Failed to generate token", "error", err)
		return er.InternalError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(models.AuthResponse{
		Token: token,
		User:  user,
	})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req models.LoginRequest

	if err := c.BodyParser(&req); err != nil {
		h.logger.Error("Failed to parse JSON", "error", err)
		return er.InvalidJSON(c)
	}

	if errorsMap := validation.ValidateRequest(req); errorsMap != nil {
		h.logger.Error("Validation failed", "errors", errorsMap)
		return er.SendValidationErrors(c, errorsMap)
	}

	user, err := h.service.GetByEmail(c.UserContext(), req.Email)
	if err != nil {
		if errors.Is(err, er.ErrUserNotFound) {
			return er.Unauthorized(c, er.ErrInvalidCredentials.Error())
		}
		h.logger.Error("Failed to get user", "error", err)
		return er.InternalError(c, err)
	}

	if !h.service.ValidatePassword(user, req.Password) {
		return er.Unauthorized(c, er.ErrInvalidCredentials.Error())
	}

	token, err := h.jwtService.GenerateToken(user.ID)
	if err != nil {
		h.logger.Error("Failed to generate token", "error", err)
		return er.InternalError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(models.AuthResponse{
		Token: token,
		User:  user,
	})
}

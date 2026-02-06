package errors

import (
	"github.com/gofiber/fiber/v2"
)

type ErrorResponse struct {
	Error            string            `json:"error,omitempty"`
	ValidationErrors map[string]string `json:"validation_errors,omitempty"`
}

func SendError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(ErrorResponse{Error: message})
}

func SendErrorFromErr(c *fiber.Ctx, status int, err error) error {
	if err == nil {
		return c.Status(status).JSON(ErrorResponse{Error: "unknown error"})
	}
	return c.Status(status).JSON(ErrorResponse{Error: err.Error()})
}

func SendValidationErrors(c *fiber.Ctx, errors map[string]string) error {
	return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{ValidationErrors: errors})
}

func BadRequest(c *fiber.Ctx, message string) error {
	return SendError(c, fiber.StatusBadRequest, message)
}

func BadRequestErr(c *fiber.Ctx, err error) error {
	return SendErrorFromErr(c, fiber.StatusBadRequest, err)
}

func NotFound(c *fiber.Ctx, message string) error {
	return SendError(c, fiber.StatusNotFound, message)
}

func InternalError(c *fiber.Ctx, err error) error {
	return SendError(c, fiber.StatusInternalServerError, "Internal server error")
}

func InvalidJSON(c *fiber.Ctx) error {
	return BadRequest(c, "Invalid JSON format")
}

func Unauthorized(c *fiber.Ctx, message string) error {
	return SendError(c, fiber.StatusUnauthorized, message)
}

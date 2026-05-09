package users

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(
	router fiber.Router,
	handler *Handler,
	logger *slog.Logger,
) {
	userGroup := router.Group("/users")
	userGroup.Get("/:id", handler.GetById)
	userGroup.Post("/", handler.Create)
	userGroup.Patch("/:id", handler.Update)
	userGroup.Delete("/:id", handler.DeleteById)
}

package users

import (
	"log/slog"
	
	"routePlanner/internal/auth"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(
	router fiber.Router,
	handler *Handler,
	authHandler *AuthHandler,
	logger *slog.Logger,
	jwtService *auth.JWTService,
) {
	authGroup := router.Group("/auth")
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/login", authHandler.Login)

	userGroup := router.Group("/users")
	userGroup.Get("/:id", handler.GetById)
	userGroup.Post("/", handler.Create)
	userGroup.Patch("/:id", handler.Update)
	userGroup.Delete("/:id", handler.DeleteById)
}

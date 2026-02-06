package places

import (
	"fmt"
	"log/slog"
	"time"

	"routePlanner/internal/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func RegisterRoutes(
	router fiber.Router,
	handler *Handler,
	logger *slog.Logger,
	storage fiber.Storage,
	placeCreateLimit int,
	jwtService *auth.JWTService,
) {
	placeGroup := router.Group("/places")

	placeCreateLimiter := limiter.New(limiter.Config{
		Storage:    storage,
		Max:        placeCreateLimit,
		Expiration: 1 * time.Hour,
		KeyGenerator: func(c *fiber.Ctx) string {
			if userID := c.Locals("user_id"); userID != nil {
				return fmt.Sprintf("%v:create_place", userID)
			}
			return c.IP() + ":create_place"
		},
		LimitReached: func(c *fiber.Ctx) error {
			logger.Warn("Hourly limit exceeded", "user_id", c.Locals("user_id"))
			return c.Status(fiber.StatusTooManyRequests).SendString("You have personally exceeded your hourly limit for creating reports.")
		},
	})

	placeGroup.Get("/:id", handler.GetById)
	placeGroup.Post("/", auth.JWTMiddleware(jwtService), placeCreateLimiter, handler.Create)
	placeGroup.Patch("/:id", handler.Update)
	placeGroup.Delete("/:id", handler.DeleteById)

}

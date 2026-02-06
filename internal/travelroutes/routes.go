package travelroutes

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func RegisterRoutes(router fiber.Router, handler *Handler, logger *slog.Logger, storage fiber.Storage, globalDailyLimit int, userLimit int) {
	travelroutesGroup := router.Group("/routes")

	globalDailyLimiter := limiter.New(limiter.Config{
		Storage:    storage,
		Max:        globalDailyLimit,
		Expiration: 24 * time.Hour,
		KeyGenerator: func(c *fiber.Ctx) string {
			return "global_geo_api_req_counter"
		},
		LimitReached: func(c *fiber.Ctx) error {
			logger.Error("Daily limit exceeded", "user_id", c.Locals("user_id"))
			return c.Status(fiber.StatusTooManyRequests).SendString("Sorry, the application-wide daily limit for this feature has been reached.")
		},
	})

	perUserSlidingLimiter := limiter.New(limiter.Config{
		Storage:           storage,
		Max:               userLimit,
		Expiration:        30 * time.Second,
		LimiterMiddleware: limiter.SlidingWindow{},
		KeyGenerator: func(c *fiber.Ctx) string {
			if userID := c.Locals("user_id"); userID != nil {
				return fmt.Sprintf("%v:build_route", userID)
			}
			return c.IP() + ":build_route"
		},
		LimitReached: func(c *fiber.Ctx) error {
			logger.Error("Rate limit exceeded", "user_id", c.Locals("user_id"))
			return c.Status(fiber.StatusTooManyRequests).SendString("You have personally exceeded your rate limit for this feature. Wait for 30 seconds and try again.")
		},
	})

	travelroutesGroup.Use(perUserSlidingLimiter)

	travelroutesGroup.Post("/analyze", handler.AnalyzeRequest)
	travelroutesGroup.Post("/build", globalDailyLimiter, handler.BuildRoute)
}

package auth

import (
	"strings"

	er "routePlanner/internal/errors"

	"github.com/gofiber/fiber/v2"
)

func JWTMiddleware(jwtService *JWTService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return er.Unauthorized(c, "Missing authorization header")
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return er.Unauthorized(c, "Invalid authorization header format")
		}

		tokenString := parts[1]
		userID, err := jwtService.ParseToken(tokenString)
		if err != nil {
			return er.Unauthorized(c, "Invalid or expired token")
		}

		c.Locals("user_id", userID)
		return c.Next()
	}
}

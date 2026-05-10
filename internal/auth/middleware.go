package auth

import (
	"strings"

	er "routePlanner/internal/errors"

	"github.com/gofiber/fiber/v2"
)

func JWTMiddleware(oidcService *OIDCService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var tokenString string

		authHeader := c.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				tokenString = parts[1]
			}
		}

		if tokenString == "" {
			tokenString = c.Cookies("auth_token")
		}

		if tokenString == "" {
			return er.Unauthorized(c, "Missing authorization")
		}

		userID, _, err := oidcService.ValidateToken(tokenString)
		if err != nil {
			return er.Unauthorized(c, "Invalid or expired token")
		}

		c.Locals("user_id", userID)
		return c.Next()
	}
}

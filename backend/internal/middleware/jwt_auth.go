package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/service"
)

func JwtAuth(svc *service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "missing Authorization header"))
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid Authorization header format"))
		}

		tokenStr := parts[1]
		user, err := svc.ValidateAccessToken(tokenStr)
		if err != nil {
			logger.Warn().Str("component", "auth").Err(err).Msg("invalid access token")
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid or expired token"))
		}

		c.Locals("user", user)
		return c.Next()
	}
}

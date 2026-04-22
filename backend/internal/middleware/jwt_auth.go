package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/repo"
	"backend/internal/service"
)

func JwtAuth(svc *service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
				tokenStr := parts[1]
				user, err := svc.ValidateAccessToken(tokenStr)
				if err == nil {
					c.Locals("user", user)
					return c.Next()
				}
				logger.Warn().Str("component", "auth").Err(err).Msg("invalid JWT access token")
			}
		}

		// If JWT is missing/invalid, check for user_access_token header
		userAccessToken := c.Get("user_access_token")
		if userAccessToken != "" {
			// This will be handled by the UserAccessToken middleware which is injected separately
			// We just pass through here so the next middleware can handle it
			return c.Next()
		}

		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "missing or invalid authentication"))
	}
}

// UserAccessTokenAuth validates the user_access_token header and sets user locals.
// This should be applied after JwtAuth as a fallback authentication method.
func UserAccessTokenAuth(userAccessTokenRepo *repo.UserAccessTokenRepo, userRepo *repo.UserRepo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip if user is already authenticated (JWT succeeded)
		if c.Locals("user") != nil {
			return c.Next()
		}

		token := c.Get("user_access_token")
		if token == "" {
			return c.Next()
		}

		// Look up the token
		accessToken, err := userAccessTokenRepo.FindByToken(c.Context(), token)
		if err != nil {
			logger.Warn().Str("component", "auth").Err(err).Msg("invalid user_access_token")
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid user access token"))
		}

		// Resolve the user
		user, err := userRepo.FindByID(c.Context(), accessToken.OwnerID)
		if err != nil {
			logger.Error().Str("component", "auth").Err(err).Int64("userId", accessToken.OwnerID).Msg("failed to resolve user from access token")
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "failed to resolve user"))
		}

		c.Locals("user", &repo.AuthUser{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
		})
		return c.Next()
	}
}

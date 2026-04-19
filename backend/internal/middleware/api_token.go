package middleware

import (
	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/repo"
)

func ApiToken(channelApiRepo *repo.ChannelApiRepo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Get("api_access_token")
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "missing api_access_token header"))
		}

		channelApi, err := channelApiRepo.FindByApiToken(c.Context(), token)
		if err != nil {
			logger.Warn().Str("component", "api-token").Err(err).Msg("invalid api_access_token")
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid api_access_token"))
		}

		if !repo.CompareApiTokenConstantTime(channelApi.ApiToken, token) {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid api_access_token"))
		}

		account, err := channelApiRepo.FindAccountByChannelID(c.Context(), channelApi.ID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to find account for channel"))
		}

		c.Locals("channelApi", channelApi)
		c.Locals("inboxId", channelApi.ID)
		c.Locals("account", account)
		c.Locals("accountId", account.ID)

		return c.Next()
	}
}

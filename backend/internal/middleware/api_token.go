package middleware

import (
	"github.com/gofiber/fiber/v2"

	"backend/internal/crypto"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/repo"
)

func APIToken(channelAPIRepo *repo.ChannelAPIRepo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Get("api_access_token")
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "missing api_access_token header"))
		}

		tokenHash := crypto.HashLookup(token)
		channelAPI, err := channelAPIRepo.FindByAPITokenHash(c.Context(), tokenHash)
		if err != nil {
			logger.Warn().Str("component", "api-token").Err(err).Msg("invalid api_access_token")
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid api_access_token"))
		}

		account, err := channelAPIRepo.FindAccountByChannelID(c.Context(), channelAPI.ID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to find account for channel"))
		}

		c.Locals("channelAPI", channelAPI)
		c.Locals("inboxId", channelAPI.ID)
		c.Locals("account", account)
		c.Locals("accountId", account.ID)

		return c.Next()
	}
}

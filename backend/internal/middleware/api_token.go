package middleware

import (
	"github.com/gofiber/fiber/v2"

	"backend/internal/crypto"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/repo"
)

// ApiToken authenticates provider (Channel::Api) requests by looking up the
// SHA-256 hash of the provided `api_access_token` header. The plaintext token
// is never stored — only the hash in channels_api.api_token_hash.
func ApiToken(channelApiRepo *repo.ChannelAPIRepo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Get("api_access_token")
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "missing api_access_token header"))
		}

		tokenHash := crypto.HashLookup(token)
		channelApi, err := channelApiRepo.FindByApiTokenHash(c.Context(), tokenHash)
		if err != nil {
			logger.Warn().Str("component", "api-token").Err(err).Msg("invalid api_access_token")
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

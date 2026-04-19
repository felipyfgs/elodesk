package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
)

func HmacOptional() fiber.Handler {
	return func(c *fiber.Ctx) error {
		channelApi, ok := c.Locals("channelApi").(*model.ChannelApi)
		if !ok {
			return c.Next()
		}

		if !channelApi.HmacMandatory {
			return c.Next()
		}

		hmacHeader := c.Get("X-Chatwoot-Hmac-Sha256")
		if hmacHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "HMAC signature is mandatory"))
		}

		body := c.Body()

		mac := hmac.New(sha256.New, []byte(channelApi.HmacToken))
		mac.Write(body)
		expectedMAC := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(hmacHeader), []byte(expectedMAC)) {
			logger.Warn().Str("component", "hmac").Msg("HMAC verification failed")
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid HMAC signature"))
		}

		return c.Next()
	}
}

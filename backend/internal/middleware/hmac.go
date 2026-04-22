package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/gofiber/fiber/v2"

	"backend/internal/crypto"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
)

func ValidIdentifierHash(channelApi *model.ChannelAPI, cipher *crypto.Cipher, identifier, providedHash string) bool {
	if providedHash == "" {
		return !channelApi.HmacMandatory
	}

	hmacKey, err := cipher.Decrypt(channelApi.HmacToken)
	if err != nil {
		logger.Error().Str("component", "hmac").Err(err).Msg("failed to decrypt hmac token for identifier_hash")
		return false
	}

	mac := hmac.New(sha256.New, []byte(hmacKey))
	mac.Write([]byte(identifier))
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(providedHash), []byte(expected))
}

// HmacOptional validates the X-Chatwoot-Hmac-Sha256 header ONLY when the
// channel has `hmac_mandatory=true`. The signing key is decrypted on the fly
// from the channel row (AES-GCM ciphertext in hmac_token).
func HmacOptional(cipher *crypto.Cipher) fiber.Handler {
	return func(c *fiber.Ctx) error {
		channelApi, ok := c.Locals("channelApi").(*model.ChannelAPI)
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

		hmacKey, err := cipher.Decrypt(channelApi.HmacToken)
		if err != nil {
			logger.Error().Str("component", "hmac").Err(err).Msg("failed to decrypt hmac token")
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "server misconfiguration"))
		}

		mac := hmac.New(sha256.New, []byte(hmacKey))
		mac.Write(c.Body())
		expected := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(hmacHeader), []byte(expected)) {
			logger.Warn().Str("component", "hmac").Msg("HMAC verification failed")
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid HMAC signature"))
		}

		return c.Next()
	}
}

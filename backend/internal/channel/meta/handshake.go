package meta

import (
	"github.com/gofiber/fiber/v2"
)

// HandleVerifyChallenge responds to the Meta webhook verification GET request.
// Meta sends hub.mode=subscribe, hub.verify_token, and hub.challenge; if the
// token matches, we echo back the challenge as text/plain.
func HandleVerifyChallenge(c *fiber.Ctx, expectedToken string) error {
	if c.Query("hub.mode") != "subscribe" {
		return c.Status(fiber.StatusUnauthorized).SendString("invalid hub.mode")
	}
	if c.Query("hub.verify_token") != expectedToken {
		return c.Status(fiber.StatusUnauthorized).SendString("verify_token mismatch")
	}
	c.Set("Content-Type", "text/plain")
	return c.SendString(c.Query("hub.challenge"))
}

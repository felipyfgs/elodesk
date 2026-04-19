package middleware

import (
	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/model"
)

func RolesRequired(roles ...model.Role) fiber.Handler {
	allowed := make(map[model.Role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("role").(model.Role)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", "role not found in context"))
		}

		if !allowed[role] {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", "insufficient permissions"))
		}

		return c.Next()
	}
}

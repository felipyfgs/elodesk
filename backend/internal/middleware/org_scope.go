package middleware

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/model"
	"backend/internal/repo"
)

func OrgScope(accountRepo *repo.AccountRepo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authUser, ok := c.Locals("user").(*repo.AuthUser)
		if !ok || authUser == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "not authenticated"))
		}

		accountIDStr := c.Params("aid")
		if accountIDStr == "" {
			accountIDStr = c.Get("X-Account-Id")
		}
		if accountIDStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "account id is required"))
		}

		accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid account id"))
		}

		account, err := accountRepo.FindByID(c.Context(), accountID)
		if err != nil {
			if repo.IsErrNotFound(err) {
				return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "account not found"))
			}
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
		}

		if account.Status == model.AccountStatusSuspended {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", "account is suspended"))
		}

		au, err := accountRepo.FindAccountUser(c.Context(), accountID, authUser.ID)
		if err != nil {
			if repo.IsErrNotFound(err) {
				return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", "you do not have access to this account"))
			}
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
		}

		c.Locals("account", account)
		c.Locals("accountId", accountID)
		c.Locals("role", au.Role)
		c.Locals("accountUser", au)

		return c.Next()
	}
}

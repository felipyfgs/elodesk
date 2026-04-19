package server

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"backend/internal/model"
	"backend/internal/repo"
)

func CurrentUser(c *fiber.Ctx) *repo.AuthUser {
	u, ok := c.Locals("user").(*repo.AuthUser)
	if !ok {
		return nil
	}
	return u
}

func CurrentAccountID(c *fiber.Ctx) int64 {
	id, ok := c.Locals("accountId").(int64)
	if !ok {
		return 0
	}
	return id
}

func CurrentRole(c *fiber.Ctx) model.Role {
	r, ok := c.Locals("role").(model.Role)
	if !ok {
		return model.RoleAgent
	}
	return r
}

func ParsePathID(c *fiber.Ctx, param string) (int64, error) {
	return strconv.ParseInt(c.Params(param), 10, 64)
}

package server

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/repo"
)

var (
	ErrNotFound  = errors.New("resource not found")
	ErrConflict  = repo.ErrConflict
	ErrForbidden = repo.ErrForbidden
)

func NotFoundHandler(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "the requested resource was not found"))
}

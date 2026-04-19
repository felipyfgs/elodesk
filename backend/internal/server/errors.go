package server

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
)

var (
	ErrNotFound  = errors.New("resource not found")
	ErrConflict  = errors.New("resource already exists")
	ErrForbidden = errors.New("access denied")
)

func NotFoundHandler(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "the requested resource was not found"))
}

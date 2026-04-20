package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/repo"
	"backend/internal/service"
)

type UserProfileHandler struct {
	svc *service.UserProfileService
}

func NewUserProfileHandler(svc *service.UserProfileService) *UserProfileHandler {
	return &UserProfileHandler{svc: svc}
}

func (h *UserProfileHandler) UpdateProfile(c *fiber.Ctx) error {
	authUser, ok := c.Locals("user").(*repo.AuthUser)
	if !ok || authUser == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "not authenticated"))
	}

	pathID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid user id"))
	}

	if pathID != authUser.ID {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", "forbidden"))
	}

	var req dto.UpdateProfileReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	accountID, _ := c.Locals("accountId").(int64)

	user, err := h.svc.UpdateProfile(c.Context(), authUser.ID, service.UpdateProfileInput{
		Name:            req.Name,
		Email:           req.Email,
		AvatarURL:       req.AvatarURL,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
		AccountID:       accountID,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCurrentPassword):
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid_current_password"))
		case errors.Is(err, service.ErrMissingCurrentPassword):
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "current_password_required"))
		case errors.Is(err, service.ErrInvalidAvatarPath):
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid_avatar_path"))
		case errors.Is(err, repo.ErrUserEmailExists):
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResp("Conflict", "email_already_registered"))
		default:
			logger.Error().Str("component", "user_profile").Err(err).Msg("failed to update profile")
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to update profile"))
		}
	}

	return c.JSON(dto.SuccessResp(fiber.Map{
		"id":        user.ID,
		"email":     user.Email,
		"name":      user.Name,
		"avatarUrl": user.AvatarURL,
		"createdAt": user.CreatedAt,
		"updatedAt": user.UpdatedAt,
	}))
}

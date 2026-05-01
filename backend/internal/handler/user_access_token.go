package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/repo"
)

type UserAccessTokenHandler struct {
	userAccessTokenRepo *repo.UserAccessTokenRepo
}

func NewUserAccessTokenHandler(userAccessTokenRepo *repo.UserAccessTokenRepo) *UserAccessTokenHandler {
	return &UserAccessTokenHandler{userAccessTokenRepo: userAccessTokenRepo}
}

// GetAccessToken godoc
//
//	@Summary		Get user access token
//	@Description	Returns the persistent access token for the authenticated user
//	@Tags			profile
//	@Produce		json
//	@Param			aid	path		int					true	"Account ID"
//	@Success		200	{object}	dto.APIResponse		"token"
//	@Failure		401	{object}	dto.APIError
//	@Failure		403	{object}	dto.APIError
//	@Failure		500	{object}	dto.APIError
//	@Router			/accounts/{aid}/profile/access_token [get]
//	@Security		BearerAuth
func (h *UserAccessTokenHandler) GetAccessToken(c *fiber.Ctx) error {
	authUser, ok := c.Locals("user").(*repo.AuthUser)
	if !ok || authUser == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "not authenticated"))
	}

	token, err := h.userAccessTokenRepo.FindByOwner(c.Context(), "User", authUser.ID)
	if err != nil {
		if errors.Is(err, repo.ErrUserAccessTokenNotFound) {
			token, err = h.userAccessTokenRepo.Create(c.Context(), "User", authUser.ID)
			if err != nil {
				logger.Error().Str("component", "user_access_token").Err(err).Int64("userId", authUser.ID).Msg("failed to create access token")
				return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create access token"))
			}
		} else {
			logger.Error().Str("component", "user_access_token").Err(err).Int64("userId", authUser.ID).Msg("failed to get access token")
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to get access token"))
		}
	}

	return c.JSON(dto.SuccessResp(fiber.Map{"token": token.Token}))
}

// ResetAccessToken godoc
//
//	@Summary		Reset user access token
//	@Description	Regenerates the persistent access token, invalidating the previous one
//	@Tags			profile
//	@Produce		json
//	@Param			aid	path		int					true	"Account ID"
//	@Success		200	{object}	dto.APIResponse		"token"
//	@Failure		401	{object}	dto.APIError
//	@Failure		403	{object}	dto.APIError
//	@Failure		500	{object}	dto.APIError
//	@Router			/accounts/{aid}/profile/access_token/reset [post]
//	@Security		BearerAuth
func (h *UserAccessTokenHandler) ResetAccessToken(c *fiber.Ctx) error {
	authUser, ok := c.Locals("user").(*repo.AuthUser)
	if !ok || authUser == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "not authenticated"))
	}

	token, err := h.userAccessTokenRepo.Regenerate(c.Context(), "User", authUser.ID)
	if err != nil {
		logger.Error().Str("component", "user_access_token").Err(err).Int64("userId", authUser.ID).Msg("failed to reset access token")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to reset access token"))
	}

	return c.JSON(dto.SuccessResp(fiber.Map{"token": token.Token}))
}

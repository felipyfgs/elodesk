package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/service"
)

type PasswordRecoveryHandler struct {
	svc *service.PasswordRecoveryService
}

func NewPasswordRecoveryHandler(svc *service.PasswordRecoveryService) *PasswordRecoveryHandler {
	return &PasswordRecoveryHandler{svc: svc}
}

// @Summary Request password reset
// @Description Sends a password reset email (always returns 200)
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.ForgotReq true "Email"
// @Success 200 {object} dto.ForgotResp
// @Router /api/v1/auth/forgot [post]
func (h *PasswordRecoveryHandler) Forgot(c *fiber.Ctx) error {
	var req dto.ForgotReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	if err := h.svc.RequestReset(c.Context(), req.Email); err != nil {
		logger.Error().Str("component", "auth").Err(err).Msg("password reset request failed")
		// Still return 200 to avoid leaking email existence.
	}

	return c.JSON(dto.SuccessResp(dto.ForgotResp{Status: "sent"}))
}

// @Summary Validate reset token
// @Description Checks if a password reset token is valid
// @Tags auth
// @Produce json
// @Param token path string true "Reset token"
// @Success 200 {object} dto.ResetValidateResp
// @Failure 404 {object} dto.APIError
// @Router /api/v1/auth/reset/{token}/validate [get]
func (h *PasswordRecoveryHandler) ValidateResetToken(c *fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "token is required"))
	}

	valid, err := h.svc.ValidateToken(c.Context(), token)
	if err != nil {
		logger.Error().Str("component", "auth").Err(err).Msg("validate reset token failed")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
	}

	if !valid {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "invalid_or_expired_token"))
	}

	return c.JSON(dto.SuccessResp(dto.ResetValidateResp{Valid: true}))
}

// @Summary Reset password
// @Description Resets password using a valid token
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.ResetReq true "Reset data"
// @Success 200
// @Failure 404 {object} dto.APIError
// @Router /api/v1/auth/reset [post]
func (h *PasswordRecoveryHandler) Reset(c *fiber.Ctx) error {
	var req dto.ResetReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	if err := h.svc.ResetPassword(c.Context(), req.Token, req.NewPassword); err != nil {
		if errors.Is(err, service.ErrResetTokenInvalid) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "invalid_or_expired_token"))
		}
		logger.Error().Str("component", "auth").Err(err).Msg("password reset failed")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
	}

	return c.JSON(dto.SuccessResp(nil))
}

package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/repo"
	"backend/internal/service"
)

type MfaHandler struct {
	mfaSvc  *service.MfaService
	authSvc *service.AuthService
}

func NewMfaHandler(mfaSvc *service.MfaService, authSvc *service.AuthService) *MfaHandler {
	return &MfaHandler{mfaSvc: mfaSvc, authSvc: authSvc}
}

// @Summary Setup MFA
// @Description Generates a TOTP secret for MFA setup
// @Tags auth
// @Produce json
// @Success 200 {object} dto.MfaSetupResp
// @Failure 401 {object} dto.APIError
// @Router /api/v1/auth/mfa/setup [post]
func (h *MfaHandler) Setup(c *fiber.Ctx) error {
	authUser, ok := c.Locals("user").(*repo.AuthUser)
	if !ok || authUser == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "not authenticated"))
	}

	result, err := h.mfaSvc.Setup(c.Context(), authUser.ID)
	if err != nil {
		logger.Error().Str("component", "auth").Err(err).Msg("mfa setup failed")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
	}

	return c.JSON(dto.SuccessResp(dto.MfaSetupResp{
		OTPAuthURI: result.OTPAuthURI,
		Secret:     result.Secret,
	}))
}

// @Summary Enable MFA
// @Description Validates TOTP code and enables MFA, returns recovery codes
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.MfaEnableReq true "TOTP code"
// @Success 200 {object} dto.MfaEnableResp
// @Failure 401 {object} dto.APIError
// @Failure 400 {object} dto.APIError
// @Router /api/v1/auth/mfa/enable [post]
func (h *MfaHandler) Enable(c *fiber.Ctx) error {
	authUser, ok := c.Locals("user").(*repo.AuthUser)
	if !ok || authUser == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "not authenticated"))
	}

	var req dto.MfaEnableReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	result, err := h.mfaSvc.Enable(c.Context(), authUser.ID, req.Code)
	if err != nil {
		if errors.Is(err, service.ErrMfaInvalidCode) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid mfa code"))
		}
		if errors.Is(err, service.ErrMfaNotSetup) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "mfa not set up"))
		}
		logger.Error().Str("component", "auth").Err(err).Msg("mfa enable failed")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
	}

	return c.JSON(dto.SuccessResp(dto.MfaEnableResp{
		RecoveryCodes: result.RecoveryCodes,
	}))
}

// @Summary Verify MFA
// @Description Verifies TOTP or recovery code and returns JWT pair
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.MfaVerifyReq true "MFA token and code"
// @Success 200 {object} dto.LoginResp
// @Failure 401 {object} dto.APIError
// @Router /api/v1/auth/mfa/verify [post]
func (h *MfaHandler) Verify(c *fiber.Ctx) error {
	var req dto.MfaVerifyReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	result, err := h.mfaSvc.Verify(c.Context(), req.MfaToken, req.Code)
	if err != nil {
		if errors.Is(err, service.ErrMfaInvalidCode) {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid mfa code"))
		}
		logger.Error().Str("component", "auth").Err(err).Msg("mfa verify failed")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
	}

	// Issue JWT pair after successful MFA verification.
	loginResult, err := h.authSvc.IssueTokenPair(c.Context(), result.UserID)
	if err != nil {
		logger.Error().Str("component", "auth").Err(err).Msg("failed to issue token pair after mfa verify")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
	}

	return c.JSON(dto.SuccessResp(dto.LoginResp{
		User:         userToResp(loginResult.User),
		Account:      accountToResp(loginResult.Account),
		AccessToken:  loginResult.AccessToken,
		RefreshToken: loginResult.RefreshToken,
	}))
}

// @Summary Disable MFA
// @Description Disables MFA and clears all recovery codes
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.MfaDisableReq true "Current password"
// @Success 200
// @Failure 401 {object} dto.APIError
// @Failure 400 {object} dto.APIError
// @Router /api/v1/auth/mfa/disable [post]
func (h *MfaHandler) Disable(c *fiber.Ctx) error {
	authUser, ok := c.Locals("user").(*repo.AuthUser)
	if !ok || authUser == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "not authenticated"))
	}

	var req dto.MfaDisableReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	if err := h.mfaSvc.Disable(c.Context(), authUser.ID, req.CurrentPassword); err != nil {
		if errors.Is(err, service.ErrMfaInvalidPassword) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid current password"))
		}
		logger.Error().Str("component", "auth").Err(err).Msg("mfa disable failed")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
	}

	return c.JSON(dto.SuccessResp(nil))
}

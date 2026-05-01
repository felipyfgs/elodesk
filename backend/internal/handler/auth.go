package handler

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
)

var Validate = validator.New()

var errResponseSent = errors.New("response already sent")

func parseAndValidate(c *fiber.Ctx, req any) error {
	if err := c.BodyParser(req); err != nil {
		_ = c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
		return errResponseSent
	}

	if err := Validate.Struct(req); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			var msgs []string
			for _, e := range validationErrors {
				msgs = append(msgs, fmt.Sprintf("field '%s' failed on '%s'", e.Field(), e.Tag()))
			}
			_ = c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Validation Error", strings.Join(msgs, "; ")))
		} else {
			_ = c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Validation Error", err.Error()))
		}
		return errResponseSent
	}
	return nil
}

func handleServiceError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials):
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid credentials"))
	case errors.Is(err, service.ErrRefreshTokenReused):
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "refresh token reuse detected"))
	case errors.Is(err, repo.ErrUserEmailExists):
		return c.Status(fiber.StatusConflict).JSON(dto.ErrorResp("Conflict", "email already registered"))
	case errors.Is(err, repo.ErrUserNotFound):
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid credentials"))
	default:
		logger.Error().Str("component", "auth").Err(err).Msg("auth service error")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
	}
}

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// @Summary System setup status
// @Description Returns whether the system has been set up (at least one user exists)
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]bool
// @Router /api/v1/auth/setup [get]
func (h *AuthHandler) SetupStatus(c *fiber.Ctx) error {
	hasUsers, err := h.svc.HasUsers(c.Context())
	if err != nil {
		logger.Error().Str("component", "auth").Err(err).Msg("failed to check setup status")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
	}
	return c.JSON(fiber.Map{"hasUsers": hasUsers, "success": true})
}

// @Summary Register a new user
// @Description Creates a user, account, and returns JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.RegisterReq true "Registration data"
// @Success 201 {object} dto.RegisterResp
// @Failure 409 {object} dto.APIError
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req dto.RegisterReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	result, err := h.svc.Register(c.Context(), req.Email, req.Password, req.Name, req.AccountName)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.RegisterResp{
		User:         userToResp(result.User),
		Account:      accountToResp(result.Account),
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}))
}

// @Summary Login
// @Description Authenticates user and returns JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.LoginReq true "Login data"
// @Success 200 {object} dto.LoginResp
// @Failure 401 {object} dto.APIError
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req dto.LoginReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	res, err := h.svc.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		return handleServiceError(c, err)
	}

	// If MFA is required, return mfa_required response instead of JWT pair.
	if res.MfaToken != "" {
		return c.JSON(dto.SuccessResp(dto.LoginRespMfa{
			MfaRequired: true,
			MfaToken:    res.MfaToken,
		}))
	}

	return c.JSON(dto.SuccessResp(dto.LoginResp{
		User:         userToResp(res.User),
		Account:      accountToResp(res.Account),
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
	}))
}

// @Summary Refresh tokens
// @Description Rotates refresh token and returns new token pair
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.RefreshReq true "Refresh token"
// @Success 200 {object} dto.RefreshResp
// @Failure 401 {object} dto.APIError
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req dto.RefreshReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	accessToken, refreshToken, err := h.svc.Refresh(c.Context(), req.RefreshToken)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.RefreshResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}))
}

// @Summary Logout
// @Description Revokes refresh token(s)
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.LogoutReq true "Logout data"
// @Success 204
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	var req dto.LogoutReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	authUser, ok := c.Locals("user").(*repo.AuthUser)
	if !ok || authUser == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "not authenticated"))
	}

	if err := h.svc.Logout(c.Context(), authUser.ID, req.RefreshToken, req.AllDevices); err != nil {
		logger.Error().Str("component", "auth").Err(err).Msg("failed to logout")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to logout"))
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func userToResp(u *model.User) dto.UserResp {
	return dto.UserResp{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		CreatedAt: u.CreatedAt,
	}
}

func accountToResp(a *model.Account) dto.AccountResp {
	return dto.AccountResp{
		ID:   a.ID,
		Name: a.Name,
		Slug: a.Slug,
	}
}

package handler

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

type AccountHandler struct {
	repo *repo.AccountRepo
}

func NewAccountHandler(repo *repo.AccountRepo) *AccountHandler {
	return &AccountHandler{repo: repo}
}

func (h *AccountHandler) Get(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	account, err := h.repo.FindByID(c.Context(), accountID)
	if err != nil {
		if errors.Is(err, repo.ErrAccountNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "account_not_found"))
		}
		logger.Error().Str("component", "account").Err(err).Msg("failed to get account")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to get account"))
	}

	return c.JSON(dto.SuccessResp(accountToDetailResp(account)))
}

func (h *AccountHandler) Update(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req dto.UpdateAccountReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	account, err := h.repo.FindByID(c.Context(), accountID)
	if err != nil {
		if errors.Is(err, repo.ErrAccountNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "account_not_found"))
		}
		logger.Error().Str("component", "account").Err(err).Msg("failed to load account")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to update account"))
	}

	if req.Name != nil {
		account.Name = strings.TrimSpace(*req.Name)
		if account.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "name_required"))
		}
	}
	if req.Locale != nil {
		account.Locale = *req.Locale
	}
	if req.Settings != nil {
		account.Settings = req.Settings
	}

	if err := h.repo.Update(c.Context(), account); err != nil {
		logger.Error().Str("component", "account").Err(err).Msg("failed to update account")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to update account"))
	}

	return c.JSON(dto.SuccessResp(accountToDetailResp(account)))
}

func accountToDetailResp(account *model.Account) dto.AccountDetailResp {
	status := int(account.Status)
	return dto.AccountDetailResp{
		ID:               account.ID,
		Name:             account.Name,
		Slug:             account.Slug,
		Locale:           account.Locale,
		Status:           status,
		CustomAttributes: account.CustomAttributes,
		Settings:         account.Settings,
		CreatedAt:        account.CreatedAt,
		UpdatedAt:        account.UpdatedAt,
	}
}

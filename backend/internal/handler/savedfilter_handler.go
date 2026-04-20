package handler

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"backend/internal/dto"
	"backend/internal/filterquery"
	"backend/internal/logger"
	"backend/internal/repo"
	"backend/internal/service"
)

type SavedFilterHandler struct {
	svc     *service.SavedFilterService
	defRepo *repo.CustomAttributeDefinitionRepo
	pool    *pgxpool.Pool
}

func NewSavedFilterHandler(svc *service.SavedFilterService, defRepo *repo.CustomAttributeDefinitionRepo, pool *pgxpool.Pool) *SavedFilterHandler {
	return &SavedFilterHandler{svc: svc, defRepo: defRepo, pool: pool}
}

func (h *SavedFilterHandler) List(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	user, ok := c.Locals("user").(*repo.AuthUser)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "user not found"))
	}

	filterType := c.Query("filter_type", "")

	filters, err := h.svc.List(c.Context(), accountID, user.ID, filterType)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.CustomFiltersToResp(filters)))
}

func (h *SavedFilterHandler) Create(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	user, ok := c.Locals("user").(*repo.AuthUser)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "user not found"))
	}

	var req dto.CreateCustomFilterReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	f, err := h.svc.Create(c.Context(), accountID, user.ID, req.Name, req.FilterType, req.Query)
	if err != nil {
		return handleFilterError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.CustomFilterToResp(f)))
}

func (h *SavedFilterHandler) Update(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	user, ok := c.Locals("user").(*repo.AuthUser)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "user not found"))
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid filter id"))
	}

	var req dto.UpdateCustomFilterReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	f, err := h.svc.Update(c.Context(), id, accountID, user.ID, req.Name, req.FilterType, req.Query)
	if err != nil {
		return handleFilterError(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.CustomFilterToResp(f)))
}

func (h *SavedFilterHandler) Delete(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	user, ok := c.Locals("user").(*repo.AuthUser)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "user not found"))
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid filter id"))
	}

	if err := h.svc.Delete(c.Context(), id, accountID, user.ID); err != nil {
		return handleFilterError(c, err)
	}

	return c.JSON(dto.SuccessResp(map[string]string{"result": "success"}))
}

func (h *SavedFilterHandler) FilterConversations(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req dto.ApplyFilterReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	perPage := req.PerPage
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}

	customKeys, _ := h.defRepo.ListKeysByModel(c.Context(), accountID, "conversation")

	where, args, err := filterquery.BuildSQL(req.Query, "conversation", customKeys)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}

	return h.executeFilter(c, accountID, where, args, page, perPage, "conversations")
}

func (h *SavedFilterHandler) FilterContacts(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req dto.ApplyFilterReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	perPage := req.PerPage
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}

	customKeys, _ := h.defRepo.ListKeysByModel(c.Context(), accountID, "contact")

	where, args, err := filterquery.BuildSQL(req.Query, "contact", customKeys)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}

	return h.executeFilter(c, accountID, where, args, page, perPage, "contacts")
}

func (h *SavedFilterHandler) executeFilter(c *fiber.Ctx, accountID int64, where string, args []any, page, perPage int, tableName string) error {
	ctx := c.Context()

	if where == "" {
		return c.JSON(dto.SuccessResp(map[string]any{
			"meta": dto.NewMetaResp(0, page, perPage), "payload": []any{},
		}))
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE account_id = $1 AND %s", tableName, where)
	countArgs := append([]any{accountID}, args...)
	var total int
	if err := h.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		logger.Error().Str("component", "saved_filters").Err(err).Msg("failed to count filters")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to count"))
	}
	if total == 0 {
		return c.JSON(dto.SuccessResp(map[string]any{
			"meta": dto.NewMetaResp(0, page, perPage), "payload": []any{},
		}))
	}

	offset := (page - 1) * perPage
	dataQuery := fmt.Sprintf("SELECT * FROM %s WHERE account_id = $1 AND %s ORDER BY created_at DESC LIMIT %d OFFSET %d", tableName, where, perPage, offset)
	dataArgs := append([]any{accountID}, args...)

	rows, err := h.pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		logger.Error().Str("component", "saved_filters").Err(err).Msg("failed to query filters")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to query"))
	}
	defer rows.Close()

	var results []json.RawMessage
	for rows.Next() {
		val, err := rows.Values()
		if err != nil {
			logger.Warn().Str("component", "saved_filters").Err(err).Msg("failed to read row values")
			continue
		}
		colNames := rows.FieldDescriptions()
		rowMap := map[string]any{}
		for i, col := range colNames {
			if i < len(val) {
				rowMap[col.Name] = val[i]
			}
		}
		b, _ := json.Marshal(rowMap)
		results = append(results, json.RawMessage(b))
	}

	return c.JSON(dto.SuccessResp(map[string]any{
		"meta":    dto.NewMetaResp(total, page, perPage),
		"payload": results,
	}))
}

func handleFilterError(c *fiber.Ctx, err error) error {
	switch {
	case repo.IsErrNotFound(err):
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "filter not found"))
	case err == service.ErrMaxFiltersReached:
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "max_filters_reached"))
	case err == service.ErrNestedOperators:
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "nested_operators_not_supported"))
	default:
		logger.Error().Str("component", "saved_filters").Err(err).Msg("saved filters service error")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
	}
}

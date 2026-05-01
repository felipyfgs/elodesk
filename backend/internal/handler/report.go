package handler

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/repo"
)

type ReportHandler struct {
	reportsRepo *repo.ReportsRepo
	slaRepo     *repo.SLARepo
}

func NewReportHandler(reportsRepo *repo.ReportsRepo, slaRepo *repo.SLARepo) *ReportHandler {
	return &ReportHandler{reportsRepo: reportsRepo, slaRepo: slaRepo}
}

// parseRange parses ?from=&to= query params (RFC3339 dates) with a 30-day
// default window ending now when omitted.
func parseRange(c *fiber.Ctx) (time.Time, time.Time) {
	now := time.Now().UTC()
	to := now
	from := now.AddDate(0, 0, -30)
	if s := c.Query("from"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			from = t
		}
	}
	if s := c.Query("to"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			to = t
		}
	}
	return from, to
}

func (h *ReportHandler) Overview(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	from, to := parseRange(c)
	report, err := h.reportsRepo.Overview(c.Context(), accountID, from, to)
	if err != nil {
		logger.Error().Str("component", "reports").Err(err).Msg("overview failed")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed"))
	}
	return c.JSON(dto.SuccessResp(report))
}

func (h *ReportHandler) Conversations(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	from, to := parseRange(c)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "50"))
	sort := c.Query("sort", "-created_at")
	var inboxID, labelID *int64
	if s := c.Query("inbox_id"); s != "" {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			inboxID = &v
		}
	}
	if s := c.Query("label_id"); s != "" {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			labelID = &v
		}
	}

	items, total, err := h.reportsRepo.Conversations(c.Context(), repo.ConversationReportFilter{
		AccountID: accountID,
		From:      from,
		To:        to,
		InboxID:   inboxID,
		LabelID:   labelID,
		Page:      page,
		PageSize:  pageSize,
		Sort:      sort,
	})
	if err != nil {
		logger.Error().Str("component", "reports").Err(err).Msg("conversations report failed")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed"))
	}
	return c.JSON(dto.SuccessResp(fiber.Map{
		"meta":    dto.NewMetaResp(total, page, pageSize),
		"payload": items,
	}))
}

func (h *ReportHandler) Entity(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	entity := c.Params("entity")
	switch entity {
	case "agents", "inboxes", "teams", "labels":
	default:
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "unknown entity"))
	}
	from, to := parseRange(c)
	metrics, err := h.reportsRepo.EntityReport(c.Context(), accountID, entity, from, to)
	if err != nil {
		logger.Error().Str("component", "reports").Err(err).Msg("entity report failed")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed"))
	}
	return c.JSON(dto.SuccessResp(metrics))
}

func (h *ReportHandler) CSAT(c *fiber.Ctx) error {
	return c.JSON(dto.SuccessResp(fiber.Map{
		"enabled": false,
		"message": "csat not enabled",
	}))
}

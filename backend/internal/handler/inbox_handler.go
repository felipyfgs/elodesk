package handler

import (
	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
)

func handleNotFound(c *fiber.Ctx, err error) error {
	if repo.IsErrNotFound(err) {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "resource not found"))
	}
	return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
}

type InboxHandler struct {
	svc *service.InboxService
}

func NewInboxHandler(svc *service.InboxService) *InboxHandler {
	return &InboxHandler{svc: svc}
}

func (h *InboxHandler) Create(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req dto.CreateInboxReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	inbox, channelApi, err := h.svc.Provision(accountID, req.Name)
	if err != nil {
		return handleNotFound(c, err)
	}

	resp := dto.CreateInboxResp{
		InboxResp: dto.InboxResp{
			ID:          inbox.ID,
			AccountID:   inbox.AccountID,
			ChannelID:   inbox.ChannelID,
			Name:        inbox.Name,
			ChannelType: inbox.ChannelType,
			CreatedAt:   inbox.CreatedAt,
		},
		Identifier: channelApi.Identifier,
		ApiToken:   channelApi.ApiToken,
		HmacToken:  channelApi.HmacToken,
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(resp))
}

func (h *InboxHandler) List(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	inboxes, err := h.svc.ListByAccount(accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	payload := make([]dto.InboxResp, len(inboxes))
	for i := range inboxes {
		payload[i] = inboxModelToResp(&inboxes[i])
	}

	return c.JSON(dto.SuccessResp(payload))
}

func (h *InboxHandler) GetByID(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid inbox id"))
	}

	inbox, err := h.svc.GetByID(int64(id), accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(inboxModelToResp(inbox)))
}

func inboxModelToResp(i *model.Inbox) dto.InboxResp {
	return dto.InboxResp{
		ID:          i.ID,
		AccountID:   i.AccountID,
		ChannelID:   i.ChannelID,
		Name:        i.Name,
		ChannelType: i.ChannelType,
		CreatedAt:   i.CreatedAt,
	}
}

package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
)

type MessageHandler struct {
	svc             *service.MessageService
	inboxRepo       *repo.InboxRepo
	contactInboxRepo *repo.ContactInboxRepo
}

func NewMessageHandler(
	svc *service.MessageService,
	inboxRepo *repo.InboxRepo,
	contactInboxRepo *repo.ContactInboxRepo,
) *MessageHandler {
	return &MessageHandler{
		svc:             svc,
		inboxRepo:       inboxRepo,
		contactInboxRepo: contactInboxRepo,
	}
}

func (h *MessageHandler) Create(c *fiber.Ctx) error {
	channelApi, ok := c.Locals("channelApi").(*model.ChannelApi)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "channel api not found"))
	}

	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	conversationID, err := strconv.ParseInt(c.Params("conversationId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	var req dto.CreateMessageReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	inbox, err := h.inboxRepo.FindByChannelID(c.Context(), channelApi.ID)
	if err != nil {
		return handleNotFound(c, err)
	}

	contentType := model.ContentTypeText
	if req.ContentType != nil {
		contentType = model.MessageContentType(*req.ContentType)
	}

	var contentAttrs *string
	if len(req.ContentAttributes) > 0 {
		s := string(req.ContentAttributes)
		contentAttrs = &s
	}

	msg := &model.Message{
		Content:      &req.Content,
		SourceID:     req.SourceID,
		Private:      req.Private,
		ContentType:  contentType,
		ContentAttrs: contentAttrs,
	}

	created, err := h.svc.Create(c.Context(), accountID, inbox.ID, conversationID, msg)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.MessageToResp(created)))
}

func (h *MessageHandler) ListPublic(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	conversationID, err := strconv.ParseInt(c.Params("conversationId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "25"))

	filter := repo.MessageListFilter{
		ConversationID: conversationID,
		AccountID:      accountID,
		Page:           page,
		PerPage:        perPage,
	}

	messages, total, err := h.svc.ListByConversation(c.Context(), filter)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.MessageListResp{
		Meta:    dto.NewMetaResp(total, page, perPage),
		Payload: dto.MessagesToResp(messages),
	}))
}

func (h *MessageHandler) List(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	conversationID, err := strconv.ParseInt(c.Params("conversationId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "25"))

	filter := repo.MessageListFilter{
		ConversationID: conversationID,
		AccountID:      accountID,
		Page:           page,
		PerPage:        perPage,
	}

	messages, total, err := h.svc.ListByConversation(c.Context(), filter)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.MessageListResp{
		Meta:    dto.NewMetaResp(total, page, perPage),
		Payload: dto.MessagesToResp(messages),
	}))
}

func (h *MessageHandler) SoftDelete(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	messageID, err := strconv.ParseInt(c.Params("messageId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid message id"))
	}

	if err := h.svc.SoftDelete(c.Context(), messageID, accountID); err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(map[string]string{"result": "success"}))
}

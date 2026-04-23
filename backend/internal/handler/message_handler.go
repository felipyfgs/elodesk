package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
)

type MessageHandler struct {
	svc              *service.MessageService
	inboxRepo        *repo.InboxRepo
	contactInboxRepo *repo.ContactInboxRepo
	messageRepo      *repo.MessageRepo
	attachmentRepo   *repo.AttachmentRepo
	conversationRepo *repo.ConversationRepo
}

func NewMessageHandler(
	svc *service.MessageService,
	inboxRepo *repo.InboxRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	messageRepo *repo.MessageRepo,
) *MessageHandler {
	return &MessageHandler{
		svc:              svc,
		inboxRepo:        inboxRepo,
		contactInboxRepo: contactInboxRepo,
		messageRepo:      messageRepo,
	}
}

func (h *MessageHandler) SetAttachmentRepo(r *repo.AttachmentRepo) {
	h.attachmentRepo = r
}

func (h *MessageHandler) SetConversationRepo(r *repo.ConversationRepo) {
	h.conversationRepo = r
}

func (h *MessageHandler) Create(c *fiber.Ctx) error {
	channelApi, ok := c.Locals("channelApi").(*model.ChannelAPI)
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

	inbox, err := h.inboxRepo.FindByChannelID(c.Context(), channelApi.ID)
	if err != nil {
		return handleNotFound(c, err)
	}

	ct := c.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		return h.createMultipart(c, accountID, inbox.ID, conversationID)
	}

	var req dto.CreateMessageReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	sourceID := req.SourceID
	if sourceID == nil && req.EchoID != nil {
		sourceID = req.EchoID
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
		SourceID:     sourceID,
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

func (h *MessageHandler) CreateAuthenticated(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	conversationID, err := strconv.ParseInt(c.Params("conversationId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	conv, err := h.conversationRepo.FindByID(c.Context(), conversationID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	inbox, err := h.inboxRepo.FindByID(c.Context(), conv.InboxID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	var req dto.CreateMessageReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	sourceID := req.SourceID
	if sourceID == nil && req.EchoID != nil {
		sourceID = req.EchoID
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
		SourceID:     sourceID,
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

func (h *MessageHandler) createMultipart(c *fiber.Ctx, accountID, inboxID, conversationID int64) error {
	// Attachments via multipart are not supported. Clients should upload files
	// via presigned URLs first, then send the message with attachment_ids.
	if form, err := c.MultipartForm(); err == nil && len(form.File) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "attachments via multipart are not supported; use presigned upload URLs"))
	}

	content := c.FormValue("content")
	if content == "" {
		content = c.FormValue("message[content]")
	}

	sourceIDStr := c.FormValue("source_id")
	if sourceIDStr == "" {
		sourceIDStr = c.FormValue("echo_id")
	}
	var sourceID *string
	if sourceIDStr != "" {
		sourceID = &sourceIDStr
	}

	private := c.FormValue("private") == "true"

	msg := &model.Message{
		Content:     &content,
		SourceID:    sourceID,
		Private:     private,
		ContentType: model.ContentTypeText,
	}

	created, err := h.svc.Create(c.Context(), accountID, inboxID, conversationID, msg)
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

	conversationID, err := strconv.ParseInt(c.Params("conversationId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	messageID, err := strconv.ParseInt(c.Params("messageId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid message id"))
	}

	// Verify the message belongs to the requested conversation
	msg, err := h.messageRepo.FindByID(c.Context(), messageID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}
	if msg.ConversationID != conversationID {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "message not found in conversation"))
	}

	if err := h.svc.SoftDelete(c.Context(), messageID, accountID); err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(map[string]string{"result": "success"}))
}

func (h *MessageHandler) UpdatePublic(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	conversationID, err := strconv.ParseInt(c.Params("convId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	messageID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid message id"))
	}

	var req struct {
		SubmittedValues struct {
			CSATSurveyResponse struct {
				Rating         int     `json:"rating"`
				FeedbackMessage *string `json:"feedback_message"`
			} `json:"csat_survey_response"`
		} `json:"submitted_values"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}

	msg, err := h.messageRepo.FindByID(c.Context(), messageID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}
	if msg.ConversationID != conversationID {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "message not found in conversation"))
	}

	if msg.ContentType != model.ContentTypeInputEmail {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResp("Unprocessable", "message is not a CSAT survey"))
	}

	if time.Since(msg.CreatedAt) > 14*24*time.Hour {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResp("Unprocessable", "You cannot update the CSAT survey after 14 days"))
	}

	attrs := map[string]any{
		"submitted_values": map[string]any{
			"csat_survey_response": map[string]any{
				"rating":          req.SubmittedValues.CSATSurveyResponse.Rating,
				"feedback_message": req.SubmittedValues.CSATSurveyResponse.FeedbackMessage,
			},
		},
	}

	if err := h.messageRepo.UpdateContentAttributes(c.Context(), messageID, accountID, attrs); err != nil {
		logger.Error().Str("component", "messages").Err(err).Msg("failed to update content attributes")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to update message"))
	}

	return c.JSON(dto.SuccessResp(dto.MessageToResp(msg)))
}

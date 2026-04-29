package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/media"
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
	minio            *media.MinioClient
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

// SetMinio injeta o cliente MinIO usado pelo caminho multipart de criação de
// mensagens. Sem ele, anexos via multipart são rejeitados — o caminho JSON
// com presigned upload continua funcionando.
func (h *MessageHandler) SetMinio(m *media.MinioClient) {
	h.minio = m
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

	if req.Content == "" && len(req.Attachments) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "message or attachments are required"))
	}

	var contentType model.MessageContentType
	if req.ContentType != nil {
		contentType = model.MessageContentType(*req.ContentType)
	}

	messageType, err := parseMessageType(req.MessageType)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}

	contentAttrs := mergeEchoID(req.ContentAttributes, req.EchoID)

	attachments := buildAttachments(req.Attachments)

	msg := &model.Message{
		Content:         &req.Content,
		MessageType:     messageType,
		SourceID:        req.SourceID,
		Private:         req.Private,
		ContentType:     contentType,
		ContentAttrs:    contentAttrs,
		Attachments:     attachments,
		SenderContactID: req.SenderContactID,
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

	if req.Content == "" && len(req.Attachments) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "message or attachments are required"))
	}

	var contentType model.MessageContentType
	if req.ContentType != nil {
		contentType = model.MessageContentType(*req.ContentType)
	}

	contentAttrs := mergeEchoID(req.ContentAttributes, req.EchoID)

	attachments := buildAttachments(req.Attachments)

	msg := &model.Message{
		Content:      &req.Content,
		MessageType:  model.MessageOutgoing,
		SourceID:     req.SourceID,
		Private:      req.Private,
		ContentType:  contentType,
		ContentAttrs: contentAttrs,
		Attachments:  attachments,
	}
	if u, ok := c.Locals("user").(*repo.AuthUser); ok && u != nil {
		senderType := "User"
		uid := u.ID
		msg.SenderType = &senderType
		msg.SenderID = &uid
	}

	created, err := h.svc.Create(c.Context(), accountID, inbox.ID, conversationID, msg)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.MessageToResp(created)))
}

func (h *MessageHandler) createMultipart(c *fiber.Ctx, accountID, inboxID, conversationID int64) error {
	content := c.FormValue("content")
	if content == "" {
		content = c.FormValue("message[content]")
	}

	sourceIDStr := c.FormValue("source_id")
	var sourceID *string
	if sourceIDStr != "" {
		sourceID = &sourceIDStr
	}

	echoIDStr := c.FormValue("echo_id")
	var echoID *string
	if echoIDStr != "" {
		echoID = &echoIDStr
	}

	private := c.FormValue("private") == "true"

	messageType := model.MessageIncoming
	if mt := c.FormValue("message_type"); mt != "" {
		parsed, err := parseMessageType(&mt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
		}
		messageType = parsed
	}

	var rawAttrs json.RawMessage
	if contentAttrsRaw := c.FormValue("content_attributes"); contentAttrsRaw != "" {
		// Validar que é JSON antes de persistir — formato livre, mas tem que
		// fazer parse para não corromper consultas downstream.
		var parsed map[string]any
		if err := json.Unmarshal([]byte(contentAttrsRaw), &parsed); err == nil {
			rawAttrs = json.RawMessage(contentAttrsRaw)
		}
	}
	contentAttrs := mergeEchoID(rawAttrs, echoID)

	attachments, err := h.uploadMultipartAttachments(c, accountID)
	if err != nil {
		return err
	}

	msg := &model.Message{
		Content:      &content,
		MessageType:  messageType,
		SourceID:     sourceID,
		Private:      private,
		ContentType:  model.ContentTypeText,
		ContentAttrs: contentAttrs,
		Attachments:  attachments,
	}

	created, err := h.svc.Create(c.Context(), accountID, inboxID, conversationID, msg)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.MessageToResp(created)))
}

// uploadMultipartAttachments lê os arquivos enviados como `attachments[]` (ou
// `attachments`, sem colchetes — alguns clients não usam) e sobe no MinIO,
// devolvendo a lista de model.Attachment com FileKey populado. Sem MinIO
// configurado, retorna 400 — o cliente deve cair no caminho JSON com
// presigned URL.
func (h *MessageHandler) uploadMultipartAttachments(c *fiber.Ctx, accountID int64) ([]model.Attachment, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, nil
	}
	files := form.File["attachments[]"]
	if len(files) == 0 {
		files = form.File["attachments"]
	}
	if len(files) == 0 {
		return nil, nil
	}
	if h.minio == nil {
		return nil, c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResp("Service Unavailable", "media storage not configured"))
	}

	ctx, cancel := context.WithTimeout(c.Context(), 2*time.Minute)
	defer cancel()

	out := make([]model.Attachment, 0, len(files))
	for _, fh := range files {
		f, err := fh.Open()
		if err != nil {
			logger.Error().Str("component", "messages").Err(err).Str("filename", fh.Filename).Msg("open multipart file")
			return nil, c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to read upload"))
		}

		safeName := sanitizeFileName(fh.Filename)
		objectPath := fmt.Sprintf("%d/uploads/%s-%s", accountID, uuid.New().String(), safeName)

		contentType := fh.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		_, putErr := h.minio.Client().PutObject(ctx, h.minio.Bucket(), objectPath, f, fh.Size, minio.PutObjectOptions{
			ContentType: contentType,
		})
		f.Close()
		if putErr != nil {
			logger.Error().Str("component", "messages").Err(putErr).Str("objectPath", objectPath).Msg("upload attachment to minio")
			return nil, c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to upload attachment"))
		}

		key := objectPath
		ext := extractExtension(fh.Filename)
		var extPtr *string
		if ext != "" {
			extPtr = &ext
		}
		fileName := fh.Filename
		out = append(out, model.Attachment{
			AccountID: accountID,
			FileKey:   &key,
			FileName:  &fileName,
			FileType:  service.FileTypeFromMime(contentType),
			Extension: extPtr,
		})
	}
	return out, nil
}

func extractExtension(filename string) string {
	i := strings.LastIndex(filename, ".")
	if i < 0 || i == len(filename)-1 {
		return ""
	}
	return strings.ToLower(filename[i+1:])
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

	senders := h.svc.HydrateMessageSenders(c.Context(), messages, accountID)
	payload := make([]dto.MessageResp, len(messages))
	for i := range messages {
		payload[i] = dto.MessageToRespWithSender(&messages[i], senders[messages[i].ID])
	}

	return c.JSON(dto.SuccessResp(dto.MessageListResp{
		Meta:    dto.NewMetaResp(total, page, perPage),
		Payload: payload,
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

	senders := h.svc.HydrateMessageSenders(c.Context(), messages, accountID)
	payload := make([]dto.MessageResp, len(messages))
	for i := range messages {
		payload[i] = dto.MessageToRespWithSender(&messages[i], senders[messages[i].ID])
	}

	return c.JSON(dto.SuccessResp(dto.MessageListResp{
		Meta:    dto.NewMetaResp(total, page, perPage),
		Payload: payload,
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

// mergeEchoID returns a content_attributes JSON blob with `echo_id`
// injected. When echoID is nil, the original attrs are returned unchanged
// (nil if empty). Echo IDs are used by the composer for optimistic
// reconciliation — persisted here so the realtime broadcast can echo them
// back without a separate column.
func mergeEchoID(attrs json.RawMessage, echoID *string) *string {
	if echoID == nil || *echoID == "" {
		if len(attrs) == 0 {
			return nil
		}
		s := string(attrs)
		return &s
	}
	merged := map[string]any{}
	if len(attrs) > 0 {
		if err := json.Unmarshal(attrs, &merged); err != nil {
			merged = map[string]any{}
		}
	}
	merged["echo_id"] = *echoID
	b, err := json.Marshal(merged)
	if err != nil {
		return nil
	}
	s := string(b)
	return &s
}

// parseMessageType maps the wire-format string used by channel-ingest callers
// (wzap, etc.) to the internal MessageType enum. Returns the zero value when
// the field is omitted, letting MessageService.Create apply its default.
func parseMessageType(raw *string) (model.MessageType, error) {
	if raw == nil || *raw == "" {
		return 0, nil
	}
	switch strings.ToLower(strings.TrimSpace(*raw)) {
	case "incoming":
		return model.MessageIncoming, nil
	case "outgoing":
		return model.MessageOutgoing, nil
	case "activity":
		return model.MessageActivity, nil
	case "template":
		return model.MessageTemplate, nil
	default:
		return 0, errors.New("invalid message_type: must be incoming, outgoing, activity or template")
	}
}

func buildAttachments(reqs []dto.CreateAttachmentReq) []model.Attachment {
	if len(reqs) == 0 {
		return nil
	}
	out := make([]model.Attachment, 0, len(reqs))
	for _, r := range reqs {
		if r.FileKey == "" {
			continue
		}
		fileKey := r.FileKey
		att := model.Attachment{
			FileKey:  &fileKey,
			FileType: service.FileTypeFromMimeOrName(r.FileType, r.FileName),
		}
		if r.FileName != "" {
			fileName := r.FileName
			att.FileName = &fileName
		}
		// Deriva extensão do filename quando ausente — o caminho JSON não
		// preenchia, fazendo o frontend cair em previews genéricos.
		if r.FileName != "" {
			if i := strings.LastIndex(r.FileName, "."); i >= 0 && i < len(r.FileName)-1 {
				ext := strings.ToLower(r.FileName[i+1:])
				if ext != "" {
					att.Extension = &ext
				}
			}
		}
		out = append(out, att)
	}
	return out
}

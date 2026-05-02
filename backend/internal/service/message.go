package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/realtime"
	"backend/internal/repo"
)

const (
	senderTypeContact = "Contact"
	senderTypeUser    = "User"
)

var ErrMessageMissingSender = errors.New("message: missing sender")

type OnOutboundMessageCreated interface {
	OnOutboundMessageCreated(ctx context.Context, accountID, inboxID int64, msg *model.Message)
}

type OnOutboundMessageUpdated interface {
	OnMessageUpdated(ctx context.Context, accountID, inboxID int64, msg *model.Message)
}

type RealtimeNotifier interface {
	Broadcast(conversationID, accountID int64, event string, payload any)
}

type conversationStore interface {
	FindByID(ctx context.Context, id, accountID int64) (*model.Conversation, error)
	FindByIDFull(ctx context.Context, accountID, id int64) (*repo.ConversationHydrated, error)
	ToggleStatus(ctx context.Context, id, accountID int64, status model.ConversationStatus) (*model.Conversation, error)
	UpdateLastActivity(ctx context.Context, id int64, at time.Time) error
	CountUnread(ctx context.Context, id, accountID int64) (int, error)
}

type MessageService struct {
	messageRepo      *repo.MessageRepo
	attachmentRepo   *repo.AttachmentRepo
	conversationRepo conversationStore
	contactRepo      *repo.ContactRepo
	userRepo         *repo.UserRepo
	realtime         RealtimeNotifier
	onOutbound       OnOutboundMessageCreated
	onMessageUpdated OnOutboundMessageUpdated
}

func NewMessageService(messageRepo *repo.MessageRepo, attachmentRepo *repo.AttachmentRepo) *MessageService {
	return &MessageService{messageRepo: messageRepo, attachmentRepo: attachmentRepo}
}

func (s *MessageService) SetConversationRepo(r conversationStore) {
	s.conversationRepo = r
}

func (s *MessageService) SetContactRepo(r *repo.ContactRepo) {
	s.contactRepo = r
}

func (s *MessageService) SetUserRepo(r *repo.UserRepo) {
	s.userRepo = r
}

func (s *MessageService) HydrateMessageSenders(ctx context.Context, messages []model.Message, accountID int64) map[int64]*dto.MessageSenderResp {
	if len(messages) == 0 {
		return nil
	}
	contactIDs := map[int64]struct{}{}
	userIDs := map[int64]struct{}{}
	for _, m := range messages {
		if m.SenderContactID != nil && *m.SenderContactID > 0 {
			contactIDs[*m.SenderContactID] = struct{}{}
			continue
		}
		if m.SenderType == nil || m.SenderID == nil {
			continue
		}
		switch *m.SenderType {
		case senderTypeContact:
			contactIDs[*m.SenderID] = struct{}{}
		case senderTypeUser:
			userIDs[*m.SenderID] = struct{}{}
		}
	}

	contacts := map[int64]*model.Contact{}
	if s.contactRepo != nil {
		for id := range contactIDs {
			if c, err := s.contactRepo.FindByID(ctx, id, accountID); err == nil {
				contacts[id] = c
			}
		}
	}
	users := map[int64]*model.User{}
	if s.userRepo != nil {
		for id := range userIDs {
			if u, err := s.userRepo.FindByID(ctx, id); err == nil {
				users[id] = u
			}
		}
	}

	out := make(map[int64]*dto.MessageSenderResp, len(messages))
	for _, m := range messages {
		if m.SenderContactID != nil && *m.SenderContactID > 0 {
			if c, ok := contacts[*m.SenderContactID]; ok {
				out[m.ID] = contactToSenderResp(c)
				continue
			}
		}
		if m.SenderType == nil || m.SenderID == nil {
			continue
		}
		switch *m.SenderType {
		case senderTypeContact:
			if c, ok := contacts[*m.SenderID]; ok {
				out[m.ID] = contactToSenderResp(c)
			}
		case senderTypeUser:
			if u, ok := users[*m.SenderID]; ok {
				out[m.ID] = userToSenderResp(u)
			}
		}
	}
	return out
}

func contactToSenderResp(c *model.Contact) *dto.MessageSenderResp {
	r := &dto.MessageSenderResp{
		ID:        c.ID,
		Name:      c.Name,
		Type:      "contact",
		AvatarURL: c.AvatarURL,
	}
	if c.AvatarURL != nil {
		r.Thumbnail = *c.AvatarURL
	}
	return r
}

func userToSenderResp(u *model.User) *dto.MessageSenderResp {
	r := &dto.MessageSenderResp{
		ID:        u.ID,
		Name:      u.Name,
		Type:      "user",
		AvatarURL: u.AvatarURL,
	}
	if u.AvatarURL != nil {
		r.Thumbnail = *u.AvatarURL
	}
	return r
}

func (s *MessageService) SetRealtimeNotifier(n RealtimeNotifier) {
	s.realtime = n
}

func (s *MessageService) SetOnOutboundHandler(h OnOutboundMessageCreated) {
	s.onOutbound = h
}

func (s *MessageService) SetOnMessageUpdated(h OnOutboundMessageUpdated) {
	s.onMessageUpdated = h
}

func (s *MessageService) Create(ctx context.Context, accountID, inboxID, conversationID int64, msg *model.Message) (*model.Message, error) {
	msg.AccountID = accountID
	msg.InboxID = inboxID
	msg.ConversationID = conversationID
	if msg.MessageType == 0 {
		msg.MessageType = model.MessageIncoming
	}

	contentEmpty := msg.Content == nil || *msg.Content == ""
	if msg.ContentType == 0 {
		if len(msg.Attachments) > 0 && contentEmpty {
			msg.ContentType = contentTypeForAttachment(msg.Attachments[0])
		} else {
			msg.ContentType = model.ContentTypeText
		}
	}

	if err := s.resolveSender(ctx, msg); err != nil {
		return nil, err
	}

	if msg.SourceID != nil && *msg.SourceID != "" {
		existing, err := s.messageRepo.FindBySourceIDConv(ctx, *msg.SourceID, conversationID, accountID)
		if err == nil {
			return existing, nil
		}
	}

	created, err := s.messageRepo.Create(ctx, msg)
	if err != nil {
		return nil, err
	}

	if s.attachmentRepo != nil && len(msg.Attachments) > 0 {
		for i := range msg.Attachments {
			att := &msg.Attachments[i]
			att.MessageID = created.ID
			att.AccountID = accountID
			if err := s.attachmentRepo.Create(ctx, att); err != nil {
				return nil, err
			}
		}
		created.Attachments = msg.Attachments
	}

	s.bumpActivity(ctx, created)
	s.reopenIfClosed(ctx, created)

	s.broadcastMessageEvent(ctx, realtime.EventMessageCreated, created)

	if s.onOutbound != nil && created.MessageType == model.MessageOutgoing && !created.Private {
		s.onOutbound.OnOutboundMessageCreated(ctx, accountID, inboxID, created)
	}

	return created, nil
}

func (s *MessageService) resolveSender(ctx context.Context, msg *model.Message) error {
	if msg.SenderType != nil && msg.SenderID != nil {
		return nil
	}

	switch msg.MessageType {
	case model.MessageIncoming, model.MessageActivity, model.MessageOutgoing:
		if s.conversationRepo == nil {
			return ErrMessageMissingSender
		}
		conv, err := s.conversationRepo.FindByID(ctx, msg.ConversationID, msg.AccountID)
		if err != nil {
			return err
		}
		st := senderTypeContact
		id := conv.ContactID
		msg.SenderType = &st
		msg.SenderID = &id
		return nil
	}

	return ErrMessageMissingSender
}

func (s *MessageService) reopenIfClosed(ctx context.Context, msg *model.Message) {
	if msg == nil || s.conversationRepo == nil {
		return
	}
	if msg.MessageType == model.MessageActivity || msg.MessageType == model.MessageTemplate || msg.Private {
		return
	}
	conv, err := s.conversationRepo.FindByID(ctx, msg.ConversationID, msg.AccountID)
	if err != nil {
		return
	}
	if conv.Status != model.ConversationResolved && conv.Status != model.ConversationSnoozed {
		return
	}
	if _, err := s.conversationRepo.ToggleStatus(ctx, conv.ID, conv.AccountID, model.ConversationOpen); err != nil {
		logger.Warn().Str("component", "message_service").
			Int64("conversationId", conv.ID).Err(err).
			Msg("reopen conversation on inbound message")
		return
	}
	s.broadcastConversationUpdated(ctx, conv.AccountID, conv.ID)
}

func (s *MessageService) broadcastConversationUpdated(ctx context.Context, accountID, convID int64) {
	if s.realtime == nil || s.conversationRepo == nil {
		return
	}
	hydrated, err := s.conversationRepo.FindByIDFull(ctx, accountID, convID)
	if err != nil {
		logger.Warn().Str("component", "message_service").
			Int64("conversationId", convID).Err(err).
			Msg("hydrate conversation for conversation.updated broadcast")
		return
	}
	row := repo.ConversationHydratedToFullRow(hydrated)
	s.realtime.Broadcast(convID, accountID, realtime.EventConversationUpdated, dto.ConversationToRespFull(&row))
}

func (s *MessageService) bumpActivity(ctx context.Context, msg *model.Message) {
	if msg == nil {
		return
	}
	if s.conversationRepo != nil {
		if err := s.conversationRepo.UpdateLastActivity(ctx, msg.ConversationID, msg.CreatedAt); err != nil {
			logger.Warn().Str("component", "message_service").
				Int64("conversationId", msg.ConversationID).Err(err).
				Msg("update conversation last_activity_at")
		}
	}
	if s.contactRepo != nil && msg.MessageType == model.MessageIncoming &&
		msg.SenderType != nil && *msg.SenderType == senderTypeContact && msg.SenderID != nil {
		if err := s.contactRepo.UpdateLastActivity(ctx, *msg.SenderID, msg.AccountID, msg.CreatedAt); err != nil {
			logger.Warn().Str("component", "message_service").
				Int64("contactId", *msg.SenderID).Err(err).
				Msg("update contact last_activity_at")
		}
	}
}

func contentTypeForAttachment(att model.Attachment) model.MessageContentType {
	switch att.FileType {
	case model.FileTypeAudio:
		return model.ContentTypeAudio
	case model.FileTypeImage:
		return model.ContentTypeImage
	case model.FileTypeVideo:
		return model.ContentTypeVideo
	default:
		return model.ContentTypeFile
	}
}

func fileTypeFromMime(mime string) model.AttachmentFileType {
	mime = strings.ToLower(mime)
	switch {
	case strings.HasPrefix(mime, "audio/"):
		return model.FileTypeAudio
	case strings.HasPrefix(mime, "image/"):
		return model.FileTypeImage
	case strings.HasPrefix(mime, "video/"):
		return model.FileTypeVideo
	default:
		return model.FileTypeFile
	}
}

func FileTypeFromMime(mime string) model.AttachmentFileType {
	return fileTypeFromMime(mime)
}

var extToFileType = map[string]model.AttachmentFileType{
	"ogg": model.FileTypeAudio, "oga": model.FileTypeAudio, "opus": model.FileTypeAudio,
	"mp3": model.FileTypeAudio, "m4a": model.FileTypeAudio, "aac": model.FileTypeAudio,
	"wav": model.FileTypeAudio, "flac": model.FileTypeAudio, "amr": model.FileTypeAudio,
	"mp4": model.FileTypeVideo, "mov": model.FileTypeVideo, "webm": model.FileTypeVideo,
	"3gp": model.FileTypeVideo, "avi": model.FileTypeVideo, "mkv": model.FileTypeVideo,
	"jpg": model.FileTypeImage, "jpeg": model.FileTypeImage, "png": model.FileTypeImage,
	"gif": model.FileTypeImage, "webp": model.FileTypeImage, "heic": model.FileTypeImage,
	"heif": model.FileTypeImage, "svg": model.FileTypeImage,
}

func FileTypeFromMimeOrName(mime, name string) model.AttachmentFileType {
	t := fileTypeFromMime(mime)
	if t != model.FileTypeFile {
		return t
	}
	i := strings.LastIndex(name, ".")
	if i < 0 || i == len(name)-1 {
		return t
	}
	ext := strings.ToLower(name[i+1:])
	if mapped, ok := extToFileType[ext]; ok {
		return mapped
	}
	return t
}

func (s *MessageService) SoftDelete(ctx context.Context, id, accountID int64) error {
	msg, err := s.messageRepo.FindByID(ctx, id, accountID)
	if err != nil {
		return err
	}
	if err := s.messageRepo.SoftDelete(ctx, id, accountID); err != nil {
		return err
	}
	s.broadcastMessageEvent(ctx, realtime.EventMessageDeleted, msg)
	return nil
}

func (s *MessageService) UpdateStatus(ctx context.Context, id, accountID int64, status string, externalErr *string) (*model.Message, error) {
	updated, err := s.messageRepo.UpdateStatus(ctx, id, accountID, status, externalErr)
	if err != nil {
		return nil, err
	}
	s.broadcastMessageEvent(ctx, realtime.EventMessageUpdated, updated)
	if s.onMessageUpdated != nil {
		s.onMessageUpdated.OnMessageUpdated(ctx, accountID, updated.InboxID, updated)
	}
	return updated, nil
}

func (s *MessageService) broadcastMessageEvent(ctx context.Context, event string, msg *model.Message) {
	if s.realtime == nil || msg == nil {
		return
	}
	var convSummary *dto.ConversationSummaryEventDTO
	if s.conversationRepo != nil {
		conv, err := s.conversationRepo.FindByID(ctx, msg.ConversationID, msg.AccountID)
		if err != nil {
			logger.Warn().Str("component", "message_service").
				Int64("conversationId", msg.ConversationID).Err(err).
				Msg("fetch conversation for realtime event")
		} else {
			unread, cerr := s.conversationRepo.CountUnread(ctx, conv.ID, conv.AccountID)
			if cerr != nil {
				logger.Warn().Str("component", "message_service").
					Int64("conversationId", conv.ID).Err(cerr).
					Msg("count unread for realtime event")
			}
			convSummary = dto.ConversationSummaryFromModel(conv, unread)
		}
	}
	var sender *dto.MessageSenderResp
	if senders := s.HydrateMessageSenders(ctx, []model.Message{*msg}, msg.AccountID); senders != nil {
		sender = senders[msg.ID]
	}
	payload := dto.MessageToEventRespWithSender(msg, convSummary, sender)
	s.realtime.Broadcast(msg.ConversationID, msg.AccountID, event, payload)
}

func (s *MessageService) ListByConversation(ctx context.Context, filter repo.MessageListFilter) ([]model.Message, int, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 || filter.PerPage > 100 {
		filter.PerPage = 25
	}
	msgs, total, err := s.messageRepo.ListByConversation(ctx, filter)
	if err != nil || len(msgs) == 0 || s.attachmentRepo == nil {
		return msgs, total, err
	}

	ids := make([]int64, len(msgs))
	for i, m := range msgs {
		ids[i] = m.ID
	}
	byMsg, err := s.attachmentRepo.FindByMessageIDs(ctx, ids)
	if err != nil {
		return msgs, total, nil
	}
	for i := range msgs {
		if atts, ok := byMsg[msgs[i].ID]; ok {
			msgs[i].Attachments = atts
		}
	}
	return msgs, total, nil
}

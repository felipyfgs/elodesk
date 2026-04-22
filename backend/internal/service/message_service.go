package service

import (
	"context"

	"backend/internal/model"
	"backend/internal/repo"
)

type OnOutboundMessageCreated interface {
	OnOutboundMessageCreated(ctx context.Context, accountID, inboxID int64, msg *model.Message)
}

type OnOutboundMessageUpdated interface {
	OnMessageUpdated(ctx context.Context, accountID, inboxID int64, msg *model.Message)
}

type MessageService struct {
	messageRepo      *repo.MessageRepo
	onOutbound       OnOutboundMessageCreated
	onMessageUpdated OnOutboundMessageUpdated
}

func NewMessageService(messageRepo *repo.MessageRepo) *MessageService {
	return &MessageService{messageRepo: messageRepo}
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
	if msg.ContentType == 0 {
		msg.ContentType = model.ContentTypeText
	}

	created, err := s.messageRepo.Create(ctx, msg)
	if err != nil {
		return nil, err
	}

	if s.onOutbound != nil && created.MessageType == model.MessageOutgoing && !created.Private {
		s.onOutbound.OnOutboundMessageCreated(ctx, accountID, inboxID, created)
	}

	return created, nil
}

func (s *MessageService) SoftDelete(ctx context.Context, id, accountID int64) error {
	return s.messageRepo.SoftDelete(ctx, id, accountID)
}

func (s *MessageService) ListByConversation(ctx context.Context, filter repo.MessageListFilter) ([]model.Message, int, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 || filter.PerPage > 100 {
		filter.PerPage = 25
	}
	return s.messageRepo.ListByConversation(ctx, filter)
}

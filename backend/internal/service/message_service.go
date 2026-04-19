package service

import (
	"context"

	"backend/internal/model"
	"backend/internal/repo"
)

type MessageService struct {
	messageRepo *repo.MessageRepo
}

func NewMessageService(messageRepo *repo.MessageRepo) *MessageService {
	return &MessageService{messageRepo: messageRepo}
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
	return s.messageRepo.Create(ctx, msg)
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

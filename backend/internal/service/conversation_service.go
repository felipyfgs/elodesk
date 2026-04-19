package service

import (
	"context"
	"fmt"

	"backend/internal/model"
	"backend/internal/repo"
)

type ConversationService struct {
	conversationRepo *repo.ConversationRepo
	contactInboxRepo *repo.ContactInboxRepo
	contactRepo      *repo.ContactRepo
}

func NewConversationService(
	conversationRepo *repo.ConversationRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	contactRepo *repo.ContactRepo,
) *ConversationService {
	return &ConversationService{
		conversationRepo: conversationRepo,
		contactInboxRepo: contactInboxRepo,
		contactRepo:      contactRepo,
	}
}

func (s *ConversationService) Create(ctx context.Context, accountID, inboxID, contactID int64) (*model.Conversation, error) {
	ci, err := s.contactInboxRepo.FindByContactAndInbox(ctx, contactID, inboxID)
	if err != nil {
		return nil, fmt.Errorf("failed to check contact inbox: %w", err)
	}

	var contactInboxID *int64
	if ci != nil {
		contactInboxID = &ci.ID
	} else {
		contact, err := s.contactRepo.FindByID(ctx, contactID, accountID)
		if err != nil {
			return nil, err
		}
		sourceID := ""
		if contact.Identifier != nil {
			sourceID = *contact.Identifier
		}
		newCI := &model.ContactInbox{
			ContactID: contactID,
			InboxID:   inboxID,
			SourceID:  sourceID,
		}
		if err := s.contactInboxRepo.Create(ctx, newCI); err != nil {
			return nil, fmt.Errorf("failed to create contact inbox: %w", err)
		}
		contactInboxID = &newCI.ID
	}

	convo := &model.Conversation{
		AccountID:      accountID,
		InboxID:        inboxID,
		Status:         model.ConversationOpen,
		ContactID:      contactID,
		ContactInboxID: contactInboxID,
	}

	if err := s.conversationRepo.Create(ctx, convo); err != nil {
		return nil, err
	}
	return convo, nil
}

func (s *ConversationService) ToggleStatus(ctx context.Context, id, accountID int64, status model.ConversationStatus) (*model.Conversation, error) {
	return s.conversationRepo.ToggleStatus(ctx, id, accountID, status)
}

func (s *ConversationService) ListByAccount(ctx context.Context, filter repo.ConversationFilter) ([]model.Conversation, int, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 || filter.PerPage > 100 {
		filter.PerPage = 25
	}
	return s.conversationRepo.ListByAccount(ctx, filter)
}

func (s *ConversationService) GetByID(ctx context.Context, id, accountID int64) (*model.Conversation, error) {
	return s.conversationRepo.FindByID(ctx, id, accountID)
}

func (s *ConversationService) UpdateLastSeen(ctx context.Context, id int64) error {
	return s.conversationRepo.UpdateLastSeen(ctx, id)
}

package service

import (
	"context"
	"fmt"

	"backend/internal/model"
	"backend/internal/repo"
)

type ContactService struct {
	contactRepo      *repo.ContactRepo
	contactInboxRepo *repo.ContactInboxRepo
	conversationRepo *repo.ConversationRepo
}

func NewContactService(
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
) *ContactService {
	return &ContactService{
		contactRepo:      contactRepo,
		contactInboxRepo: contactInboxRepo,
		conversationRepo: conversationRepo,
	}
}

func (s *ContactService) Create(ctx context.Context, accountID int64, contact *model.Contact) (*model.Contact, error) {
	contact.AccountID = accountID

	if contact.Identifier != nil && *contact.Identifier != "" {
		existing, err := s.contactRepo.FindByIdentifier(ctx, *contact.Identifier, fmt.Sprintf("%d", accountID))
		if err == nil {
			if contact.Name != "" {
				existing.Name = contact.Name
			}
			if contact.Email != nil {
				existing.Email = contact.Email
			}
			if contact.PhoneNumber != nil {
				existing.PhoneNumber = contact.PhoneNumber
			}
			if contact.AdditionalAttrs != nil {
				existing.AdditionalAttrs = contact.AdditionalAttrs
			}
			return s.contactRepo.Update(ctx, existing.ID, accountID, &existing.Name, existing.Email, existing.PhoneNumber)
		}
		if !repo.IsErrNotFound(err) {
			return nil, fmt.Errorf("failed to check existing contact: %w", err)
		}
	}

	if err := s.contactRepo.Create(ctx, contact); err != nil {
		return nil, err
	}
	return contact, nil
}

func (s *ContactService) FindByID(ctx context.Context, id, accountID int64) (*model.Contact, error) {
	return s.contactRepo.FindByID(ctx, id, accountID)
}

func (s *ContactService) Search(ctx context.Context, accountID int64, query string, page, perPage int) ([]model.Contact, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}
	filter := repo.ContactFilter{
		AccountID: accountID,
		Query:     query,
		Page:      page,
		PerPage:   perPage,
	}
	return s.contactRepo.Search(ctx, filter)
}

func (s *ContactService) Update(ctx context.Context, id, accountID int64, name, email, phone *string) (*model.Contact, error) {
	return s.contactRepo.Update(ctx, id, accountID, name, email, phone)
}

func (s *ContactService) FindConversations(ctx context.Context, contactID, accountID int64) ([]model.Conversation, error) {
	contactIDCopy := contactID
	filter := repo.ConversationFilter{
		AccountID: accountID,
		ContactID: &contactIDCopy,
		Page:      1,
		PerPage:   1000,
	}
	convos, _, err := s.conversationRepo.ListByAccount(ctx, filter)
	return convos, err
}

func (s *ContactService) FindBySourceID(ctx context.Context, sourceID string, inboxID, accountID int64) (*model.Contact, error) {
	ci, err := s.contactInboxRepo.FindBySourceID(ctx, sourceID, inboxID)
	if err != nil {
		return nil, err
	}
	return s.contactRepo.FindByID(ctx, ci.ContactID, accountID)
}

func (s *ContactService) EnsureContactInbox(ctx context.Context, contactID, inboxID int64, sourceID string) error {
	existing, err := s.contactInboxRepo.FindByContactAndInbox(ctx, contactID, inboxID)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	ci := &model.ContactInbox{
		ContactID: contactID,
		InboxID:   inboxID,
		SourceID:  sourceID,
	}
	return s.contactInboxRepo.Create(ctx, ci)
}

package webwidget

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

type SessionService struct {
	widgetRepo       *repo.ChannelWebWidgetRepo
	contactRepo      *repo.ContactRepo
	contactInboxRepo *repo.ContactInboxRepo
	conversationRepo *repo.ConversationRepo
	jwtSvc           *VisitorJWTService
}

func NewSessionService(
	widgetRepo *repo.ChannelWebWidgetRepo,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	jwtSvc *VisitorJWTService,
) *SessionService {
	return &SessionService{
		widgetRepo:       widgetRepo,
		contactRepo:      contactRepo,
		contactInboxRepo: contactInboxRepo,
		conversationRepo: conversationRepo,
		jwtSvc:           jwtSvc,
	}
}

func generatePubsubToken() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *SessionService) CreateOrResumeSession(ctx context.Context, websiteToken string, existingClaims *VisitorClaims, ip string) (*SessionResult, error) {
	widget, err := s.widgetRepo.FindByWebsiteToken(ctx, websiteToken)
	if err != nil {
		return nil, fmt.Errorf("widget not found: %w", err)
	}

	if existingClaims != nil && existingClaims.ContactID > 0 && existingClaims.WebsiteToken == websiteToken {
		contact, err := s.contactRepo.FindByID(ctx, existingClaims.ContactID, widget.AccountID)
		if err == nil {
			var conversation *model.Conversation
			if existingClaims.ConversationID > 0 {
				conversation, err = s.conversationRepo.FindByID(ctx, existingClaims.ConversationID, widget.AccountID)
				if err != nil || conversation.Status != model.ConversationOpen {
					conversation, err = s.conversationRepo.EnsureOpen(ctx, widget.AccountID, widget.InboxID, contact.ID)
					if err != nil {
						return nil, fmt.Errorf("failed to ensure open conversation: %w", err)
					}
				}
			}
			if conversation == nil {
				conversation, err = s.conversationRepo.EnsureOpen(ctx, widget.AccountID, widget.InboxID, contact.ID)
				if err != nil {
					return nil, fmt.Errorf("failed to ensure open conversation: %w", err)
				}
			}

			pubsubToken := ""
			if conversation.PubsubToken != nil {
				pubsubToken = *conversation.PubsubToken
			}
			if pubsubToken == "" {
				pubsubToken = generatePubsubToken()
				_ = s.conversationRepo.UpdatePubsubToken(ctx, conversation.ID, &pubsubToken)
			}

			identifier := ""
			if contact.Identifier != nil {
				identifier = *contact.Identifier
			}

			jwtStr, err := s.jwtSvc.Issue(contact.ID, identifier, websiteToken, conversation.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to issue visitor jwt: %w", err)
			}

			return &SessionResult{
				ContactID:         contact.ID,
				ContactIdentifier: identifier,
				ConversationID:    conversation.ID,
				PubsubToken:       pubsubToken,
				JWT:               jwtStr,
			}, nil
		}
		logger.Info().Str("component", "webwidget.session").Msg("existing contact not found, creating new session")
	}

	return s.createNewSession(ctx, widget, ip)
}

func generateRandomID(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *SessionService) createNewSession(ctx context.Context, widget *model.ChannelWebWidget, ip string) (*SessionResult, error) {
	identifier := "anon_" + generateRandomID(16)

	meta := fmt.Sprintf(`{"browser":"unknown","ip":"%s","source":"widget"}`, ip)

	contact := &model.Contact{
		AccountID:       widget.AccountID,
		Name:            "",
		Identifier:      &identifier,
		AdditionalAttrs: &meta,
	}
	if err := s.contactRepo.Create(ctx, contact); err != nil {
		return nil, fmt.Errorf("failed to create contact: %w", err)
	}

	contactInbox := &model.ContactInbox{
		ContactID: contact.ID,
		InboxID:   widget.InboxID,
		SourceID:  identifier,
	}
	if err := s.contactInboxRepo.Create(ctx, contactInbox); err != nil {
		return nil, fmt.Errorf("failed to create contact inbox: %w", err)
	}

	pubsubToken := generatePubsubToken()
	pt := pubsubToken

	conversation := &model.Conversation{
		AccountID:      widget.AccountID,
		InboxID:        widget.InboxID,
		Status:         model.ConversationOpen,
		ContactID:      contact.ID,
		ContactInboxID: &contactInbox.ID,
		PubsubToken:    &pt,
	}
	if err := s.conversationRepo.CreateWithPubsubToken(ctx, conversation); err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	jwtStr, err := s.jwtSvc.Issue(contact.ID, identifier, widget.WebsiteToken, conversation.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to issue visitor jwt: %w", err)
	}

	return &SessionResult{
		ContactID:         contact.ID,
		ContactIdentifier: identifier,
		ConversationID:    conversation.ID,
		PubsubToken:       pubsubToken,
		JWT:               jwtStr,
	}, nil
}

package webwidget

import (
	"context"
	"fmt"

	"backend/internal/logger"
	"backend/internal/repo"

	appcrypto "backend/internal/crypto"
)

type IdentifyService struct {
	widgetRepo       *repo.ChannelWebWidgetRepo
	contactRepo      *repo.ContactRepo
	contactInboxRepo *repo.ContactInboxRepo
	conversationRepo *repo.ConversationRepo
	cipher           *appcrypto.Cipher
	jwtSvc           *VisitorJWTService
}

func NewIdentifyService(
	widgetRepo *repo.ChannelWebWidgetRepo,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	cipher *appcrypto.Cipher,
	jwtSvc *VisitorJWTService,
) *IdentifyService {
	return &IdentifyService{
		widgetRepo:       widgetRepo,
		contactRepo:      contactRepo,
		contactInboxRepo: contactInboxRepo,
		conversationRepo: conversationRepo,
		cipher:           cipher,
		jwtSvc:           jwtSvc,
	}
}

func (s *IdentifyService) Identify(ctx context.Context, claims *VisitorClaims, req *IdentifyRequest) (*IdentifyResult, error) {
	widget, err := s.widgetRepo.FindByWebsiteToken(ctx, claims.WebsiteToken)
	if err != nil {
		return nil, fmt.Errorf("widget not found: %w", err)
	}

	hmacToken, err := s.cipher.Decrypt(widget.HmacTokenCiphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt hmac token: %w", err)
	}

	verified := false
	if req.IdentifierHash != "" {
		if !VerifyIdentifierHash(hmacToken, req.Identifier, req.IdentifierHash) {
			logger.Warn().Str("component", "webwidget.identify").
				Str("identifier", req.Identifier).
				Msg("invalid identifier hash")
			return nil, fmt.Errorf("invalid_identifier_hash")
		}
		verified = true
	}

	existingContact, err := s.contactRepo.FindByIdentifier(ctx, req.Identifier, fmt.Sprintf("%d", widget.AccountID))
	if err == nil {
		if existingContact.ID != claims.ContactID {
			anonContact, err := s.contactRepo.FindByID(ctx, claims.ContactID, widget.AccountID)
			if err != nil {
				return nil, fmt.Errorf("failed to find anonymous contact: %w", err)
			}

			if err := s.mergeConversations(ctx, anonContact.ID, existingContact.ID, widget.AccountID, widget.InboxID); err != nil {
				logger.Warn().Str("component", "webwidget.identify").Err(err).Msg("failed to merge conversations")
			}
		}

		if req.Name != nil && *req.Name != "" {
			existingContact.Name = *req.Name
		}
		if req.Email != nil && *req.Email != "" {
			existingContact.Email = req.Email
		}

		contact, err := s.contactRepo.Update(ctx, existingContact.ID, widget.AccountID, &existingContact.Name, existingContact.Email, existingContact.PhoneNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to update contact: %w", err)
		}

		if verified {
			if ci, err := s.contactInboxRepo.FindByContactAndInbox(ctx, contact.ID, widget.InboxID); err == nil && ci != nil && !ci.HmacVerified {
				_ = s.contactInboxRepo.UpdateHmacVerified(ctx, ci.ID, true)
			}
		}

		identifier := ""
		if contact.Identifier != nil {
			identifier = *contact.Identifier
		}

		jwtStr, err := s.jwtSvc.Issue(contact.ID, identifier, claims.WebsiteToken, claims.ConversationID)
		if err != nil {
			return nil, fmt.Errorf("failed to issue visitor jwt: %w", err)
		}

		return &IdentifyResult{
			ContactID:         contact.ID,
			ContactIdentifier: identifier,
			Verified:          verified,
			JWT:               jwtStr,
		}, nil
	}

	anonContact, err := s.contactRepo.FindByID(ctx, claims.ContactID, widget.AccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to find anonymous contact: %w", err)
	}

	name := anonContact.Name
	if req.Name != nil {
		name = *req.Name
	}
	email := anonContact.Email
	if req.Email != nil {
		email = req.Email
	}

	contact, err := s.contactRepo.UpdateIdentifier(ctx, anonContact.ID, widget.AccountID, req.Identifier, &name, email)
	if err != nil {
		return nil, fmt.Errorf("failed to update contact identifier: %w", err)
	}

	if verified {
		if ci, err := s.contactInboxRepo.FindByContactAndInbox(ctx, contact.ID, widget.InboxID); err == nil && ci != nil {
			_ = s.contactInboxRepo.UpdateHmacVerified(ctx, ci.ID, true)
		}
	}

	identifier := ""
	if contact.Identifier != nil {
		identifier = *contact.Identifier
	}

	jwtStr, err := s.jwtSvc.Issue(contact.ID, identifier, claims.WebsiteToken, claims.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to issue visitor jwt: %w", err)
	}

	return &IdentifyResult{
		ContactID:         contact.ID,
		ContactIdentifier: identifier,
		Verified:          verified,
		JWT:               jwtStr,
	}, nil
}

func (s *IdentifyService) mergeConversations(ctx context.Context, fromContactID, toContactID, accountID, inboxID int64) error {
	conversations, _, err := s.contactRepo.ListConversationsByContactID(ctx, fromContactID, accountID, 1, 1000)
	if err != nil {
		return err
	}

	for _, c := range conversations {
		if c.InboxID == inboxID {
			_, err := s.conversationRepo.UpdateContactID(ctx, c.ID, accountID, toContactID)
			if err != nil {
				return err
			}
		}
	}

	attrs := fmt.Sprintf(`{"merged_from_contact_id":%d}`, fromContactID)
	_, _ = s.contactRepo.UpdateAdditionalAttrs(ctx, toContactID, accountID, attrs)

	return nil
}

package service

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"
	"backend/internal/realtime"
	"backend/internal/repo"
)

const (
	MaxForwardMessages = 5
	MaxForwardTargets  = 5
)

var (
	ErrForwardLimitExceeded      = errors.New("max 5 messages per dispatch")
	ErrForwardTargetsLimit       = errors.New("max 5 targets per dispatch")
	ErrForwardNoAttachments      = errors.New("messages must have content or attachments")
	ErrForwardIncompatibleTarget = errors.New("target inbox is incompatible with message attachments")
	ErrForwardNoTargets          = errors.New("no valid targets")
	ErrForwardEmptySource        = errors.New("source_message_ids is required")
	ErrForwardInvalidTarget      = errors.New("target must specify conversation_id or (contact_id + inbox_id)")
)

type ForwardTarget struct {
	ConversationID int64
	ContactID      int64
	InboxID        int64
}

type ForwardResult struct {
	Target              ForwardTarget
	Status              string // "success" or "failed"
	CreatedMessageIDs   []int64
	ConversationID      int64
	CreatedConversation bool
	Err                 error
}

type ForwardService struct {
	messageRepo      *repo.MessageRepo
	attachmentRepo   *repo.AttachmentRepo
	conversationRepo *repo.ConversationRepo
	contactInboxRepo *repo.ContactInboxRepo
	contactRepo      *repo.ContactRepo
	inboxRepo        *repo.InboxRepo
	messageSvc       *MessageService
	conversationSvc  *ConversationService
}

func NewForwardService(
	messageRepo *repo.MessageRepo,
	attachmentRepo *repo.AttachmentRepo,
	conversationRepo *repo.ConversationRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	contactRepo *repo.ContactRepo,
	inboxRepo *repo.InboxRepo,
	messageSvc *MessageService,
	conversationSvc *ConversationService,
) *ForwardService {
	return &ForwardService{
		messageRepo:      messageRepo,
		attachmentRepo:   attachmentRepo,
		conversationRepo: conversationRepo,
		contactInboxRepo: contactInboxRepo,
		contactRepo:      contactRepo,
		inboxRepo:        inboxRepo,
		messageSvc:       messageSvc,
		conversationSvc:  conversationSvc,
	}
}

func (s *ForwardService) ForwardMessages(
	ctx context.Context,
	accountID, agentID int64,
	sourceMessageIDs []int64,
	targets []ForwardTarget,
) ([]ForwardResult, error) {
	if len(sourceMessageIDs) == 0 {
		return nil, ErrForwardEmptySource
	}
	if len(sourceMessageIDs) > MaxForwardMessages {
		return nil, ErrForwardLimitExceeded
	}
	if len(targets) == 0 {
		return nil, ErrForwardNoTargets
	}
	if len(targets) > MaxForwardTargets {
		return nil, ErrForwardTargetsLimit
	}

	sourceMessages, err := s.messageRepo.FindByIDs(ctx, sourceMessageIDs, accountID)
	if err != nil {
		return nil, fmt.Errorf("load source messages: %w", err)
	}
	if len(sourceMessages) != len(sourceMessageIDs) {
		return nil, repo.ErrMessageNotFound
	}

	rootIDs := make([]int64, len(sourceMessages))
	msgIDs := make([]int64, len(sourceMessages))
	for i, msg := range sourceMessages {
		rootIDs[i] = resolveRootForwardID(msg)
		msgIDs[i] = msg.ID
	}

	attachmentsByMsg, err := s.attachmentRepo.FindByMessageIDs(ctx, msgIDs)
	if err != nil {
		return nil, fmt.Errorf("load attachments: %w", err)
	}

	for _, t := range targets {
		if t.ConversationID <= 0 && (t.ContactID <= 0 || t.InboxID <= 0) {
			return nil, ErrForwardInvalidTarget
		}
		if t.ConversationID > 0 {
			conv, err := s.conversationRepo.FindByID(ctx, t.ConversationID, accountID)
			if err != nil {
				return nil, fmt.Errorf("target conversation %d: %w", t.ConversationID, err)
			}
			inbox, err := s.inboxRepo.FindByID(ctx, conv.InboxID, accountID)
			if err != nil {
				return nil, fmt.Errorf("target inbox for conversation %d: %w", t.ConversationID, err)
			}
			if err := s.validateCompatibility(inbox.ChannelType, sourceMessages, attachmentsByMsg); err != nil {
				return nil, fmt.Errorf("conversation %d: %w", t.ConversationID, err)
			}
		} else {
			if _, err := s.contactRepo.FindByID(ctx, t.ContactID, accountID); err != nil {
				return nil, fmt.Errorf("target contact %d: %w", t.ContactID, err)
			}
			inbox, err := s.inboxRepo.FindByID(ctx, t.InboxID, accountID)
			if err != nil {
				return nil, fmt.Errorf("target inbox %d: %w", t.InboxID, err)
			}
			if err := s.validateCompatibility(inbox.ChannelType, sourceMessages, attachmentsByMsg); err != nil {
				return nil, fmt.Errorf("contact %d, inbox %d: %w", t.ContactID, t.InboxID, err)
			}
		}
	}

	results := make([]ForwardResult, 0, len(targets))
	for _, target := range targets {
		results = append(results, s.forwardToTarget(ctx, accountID, agentID, target, sourceMessages, rootIDs, attachmentsByMsg))
	}
	return results, nil
}

func (s *ForwardService) forwardToTarget(
	ctx context.Context,
	accountID, agentID int64,
	target ForwardTarget,
	sourceMessages []model.Message,
	rootIDs []int64,
	attachmentsByMsg map[int64][]model.Attachment,
) ForwardResult {
	result := ForwardResult{
		Target: target,
		Status: "failed",
	}

	conversationID, inboxID, createdConversation, err := s.resolveTargetConversation(ctx, accountID, target)
	if err != nil {
		result.Err = err
		return result
	}

	createdMessages := make([]*model.Message, 0, len(sourceMessages))
	createdIDs := make([]int64, 0, len(sourceMessages))
	for i, src := range sourceMessages {
		rootID := rootIDs[i]
		senderType := senderTypeUser
		msg := &model.Message{
			AccountID:              accountID,
			InboxID:                inboxID,
			ConversationID:         conversationID,
			MessageType:            model.MessageOutgoing,
			ContentType:            src.ContentType,
			Content:                src.Content,
			Private:                src.Private,
			Status:                 model.MessageSent,
			ContentAttrs:           src.ContentAttrs,
			SenderType:             &senderType,
			SenderID:               &agentID,
			ForwardedFromMessageID: &rootID,
		}

		created, err := s.messageRepo.Create(ctx, msg)
		if err != nil {
			result.Err = fmt.Errorf("create message %d: %w", src.ID, err)
			return result
		}
		createdIDs = append(createdIDs, created.ID)

		for _, att := range attachmentsByMsg[src.ID] {
			newAtt := model.Attachment{
				MessageID:   created.ID,
				AccountID:   accountID,
				FileType:    att.FileType,
				ExternalURL: att.ExternalURL,
				FileKey:     att.FileKey,
				FileName:    att.FileName,
				Extension:   att.Extension,
				Meta:        att.Meta,
			}
			if err := s.attachmentRepo.Create(ctx, &newAtt); err != nil {
				result.Err = fmt.Errorf("create attachment for message %d: %w", created.ID, err)
				return result
			}
			created.Attachments = append(created.Attachments, newAtt)
		}
		createdMessages = append(createdMessages, created)
	}

	for _, created := range createdMessages {
		s.messageSvc.bumpActivity(ctx, created)
		s.messageSvc.reopenIfClosed(ctx, created)
		s.messageSvc.broadcastMessageEvent(ctx, realtime.EventMessageCreated, created)

		if s.messageSvc.onOutbound != nil && created.MessageType == model.MessageOutgoing && !created.Private {
			s.messageSvc.onOutbound.OnOutboundMessageCreated(ctx, accountID, created.InboxID, created)
		}
	}

	result.Status = "success"
	result.CreatedMessageIDs = createdIDs
	result.ConversationID = conversationID
	result.CreatedConversation = createdConversation
	return result
}

func (s *ForwardService) resolveTargetConversation(ctx context.Context, accountID int64, target ForwardTarget) (conversationID, inboxID int64, createdConversation bool, err error) {
	if target.ConversationID > 0 {
		conv, ferr := s.conversationRepo.FindByID(ctx, target.ConversationID, accountID)
		if ferr != nil {
			return 0, 0, false, fmt.Errorf("find conversation %d: %w", target.ConversationID, ferr)
		}
		return conv.ID, conv.InboxID, false, nil
	}

	inboxID = target.InboxID
	ci, ferr := s.contactInboxRepo.FindByContactAndInbox(ctx, target.ContactID, target.InboxID)
	if ferr != nil {
		return 0, 0, false, fmt.Errorf("find contact inbox: %w", ferr)
	}
	if ci != nil && ci.ID > 0 {
		latest, lerr := s.conversationRepo.FindLatestByContactInbox(ctx, ci.ID, accountID)
		if lerr != nil {
			return 0, 0, false, fmt.Errorf("find latest conversation: %w", lerr)
		}
		if latest != nil {
			return latest.ID, inboxID, false, nil
		}
	} else {
		inbox, ferr := s.inboxRepo.FindByID(ctx, target.InboxID, accountID)
		if ferr != nil {
			return 0, 0, false, fmt.Errorf("find inbox %d: %w", target.InboxID, ferr)
		}
		contact, ferr := s.contactRepo.FindByID(ctx, target.ContactID, accountID)
		if ferr != nil {
			return 0, 0, false, fmt.Errorf("find contact %d: %w", target.ContactID, ferr)
		}
		sourceID, ferr := sourceIDForChannel(inbox.ChannelType, contact)
		if ferr != nil {
			return 0, 0, false, ferr
		}
		newCI := &model.ContactInbox{
			ContactID: target.ContactID,
			InboxID:   target.InboxID,
			SourceID:  sourceID,
		}
		if ferr := s.contactInboxRepo.Create(ctx, newCI); ferr != nil {
			return 0, 0, false, fmt.Errorf("create contact inbox: %w", ferr)
		}
	}

	conv, cerr := s.conversationSvc.CreateWithOpts(ctx, accountID, target.InboxID, target.ContactID, ConversationCreateOpts{})
	if cerr != nil {
		return 0, 0, false, fmt.Errorf("create conversation: %w", cerr)
	}
	return conv.ID, inboxID, true, nil
}

func sourceIDForChannel(channelType string, contact *model.Contact) (string, error) {
	switch channelType {
	case "Channel::Whatsapp", "Channel::Sms", "Channel::Twilio":
		if contact.PhoneE164 != nil && *contact.PhoneE164 != "" {
			return *contact.PhoneE164, nil
		}
		if contact.PhoneNumber != nil && *contact.PhoneNumber != "" {
			return *contact.PhoneNumber, nil
		}
		return "", fmt.Errorf("contact has no phone number for %s", channelType)
	case "Channel::Email":
		if contact.Email != nil && *contact.Email != "" {
			return *contact.Email, nil
		}
		return "", fmt.Errorf("contact has no email for %s", channelType)
	default:
		if contact.Identifier != nil && *contact.Identifier != "" {
			return *contact.Identifier, nil
		}
		return "", fmt.Errorf("contact has no identifier for %s", channelType)
	}
}

func (s *ForwardService) validateCompatibility(channelType string, sources []model.Message, attachmentsByMsg map[int64][]model.Attachment) error {
	for _, src := range sources {
		for _, att := range attachmentsByMsg[src.ID] {
			if !IsAttachmentCompatibleWithChannel(channelType, att.FileType) {
				return fmt.Errorf("%w: %s does not support %s",
					ErrForwardIncompatibleTarget,
					channelType,
					AttachmentFileTypeName(att.FileType),
				)
			}
		}
	}
	return nil
}

func resolveRootForwardID(m model.Message) int64 {
	if m.ForwardedFromMessageID != nil {
		return *m.ForwardedFromMessageID
	}
	return m.ID
}

package service

import (
	"context"
	"encoding/json"
	"fmt"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/realtime"
	"backend/internal/repo"
)

type ConversationCreateOpts struct {
	CustomAttributes     map[string]any
	AdditionalAttributes map[string]any
	Status               *model.ConversationStatus
	SnoozedUntil         *string
	AssigneeID           *int64
	TeamID               *int64
}

type ConversationNotifier interface {
	OnConversationCreated(ctx context.Context, accountID, inboxID int64, conv *model.Conversation)
	OnConversationStatusChanged(ctx context.Context, accountID, inboxID int64, conv *model.Conversation)
	OnConversationUpdated(ctx context.Context, accountID, inboxID int64, conv *model.Conversation, attributes json.RawMessage)
}

type ConversationService struct {
	conversationRepo *repo.ConversationRepo
	contactInboxRepo *repo.ContactInboxRepo
	contactRepo      *repo.ContactRepo
	slaRepo          *repo.SLARepo
	notifications    NotificationCreator
	notifier         ConversationNotifier
	realtime         RealtimeNotifier
	messageSvc       *MessageService // optional, for hydrating last_non_activity sender
}

func NewConversationService(
	conversationRepo *repo.ConversationRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	contactRepo *repo.ContactRepo,
	slaRepo *repo.SLARepo,
	notifications NotificationCreator,
) *ConversationService {
	return &ConversationService{
		conversationRepo: conversationRepo,
		contactInboxRepo: contactInboxRepo,
		contactRepo:      contactRepo,
		slaRepo:          slaRepo,
		notifications:    notifications,
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

	if s.slaRepo != nil {
		if _, err := s.slaRepo.AttachIfUnset(ctx, accountID, convo.ID); err != nil {
			logger.Warn().Str("component", "conversation").Err(err).Int64("conversation_id", convo.ID).Msg("failed to attach sla policy")
		}
	}
	return convo, nil
}

func (s *ConversationService) ToggleStatus(ctx context.Context, id, accountID int64, status model.ConversationStatus) (*model.Conversation, error) {
	convo, err := s.conversationRepo.ToggleStatus(ctx, id, accountID, status)
	if err != nil {
		return nil, err
	}
	s.broadcastUpdated(ctx, accountID, convo.ID)
	return convo, nil
}

// broadcastUpdated hydrates the conversation and pushes a
// `conversation.updated` event so connected clients can react without polling.
// Best-effort: errors are logged and swallowed so the caller's response isn't
// blocked by realtime hiccups.
func (s *ConversationService) broadcastUpdated(ctx context.Context, accountID, convoID int64) {
	if s.realtime == nil {
		return
	}
	hydrated, err := s.conversationRepo.FindByIDFull(ctx, accountID, convoID)
	if err != nil {
		logger.Warn().Str("component", "conversation").Err(err).Int64("conversation_id", convoID).Msg("failed to hydrate conversation for realtime broadcast")
		return
	}
	fullRow := repo.ConversationHydratedToFullRow(hydrated)
	resp := dto.ConversationToRespFull(&fullRow)
	s.realtime.Broadcast(convoID, accountID, realtime.EventConversationUpdated, resp)
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

func (s *ConversationService) CountMeta(ctx context.Context, accountID, currentUserID int64, inboxID *int64) (repo.ConversationMetaCounts, error) {
	return s.conversationRepo.CountByStatusAndAssignee(ctx, accountID, currentUserID, inboxID)
}

// MetaByFilter returns the flat assignee-dimension counts (mine, assigned,
// unassigned, all) honoring the same filter applied to the list endpoint.
// Used as the `meta` envelope alongside the conversations list payload.
func (s *ConversationService) MetaByFilter(ctx context.Context, filter repo.ConversationFilter) (dto.ConversationListMeta, error) {
	return s.conversationRepo.CountByFilter(ctx, filter)
}

// ListWithMeta runs the list query and counts in one go, hydrating each row
// to the Chatwoot-shape DTO. Failures hydrating individual rows fall back to
// the bare ConversationToResp shape so the page still renders.
func (s *ConversationService) ListWithMeta(ctx context.Context, filter repo.ConversationFilter) ([]dto.ConversationResp, dto.ConversationListMeta, error) {
	convos, total, err := s.ListByAccount(ctx, filter)
	if err != nil {
		return nil, dto.ConversationListMeta{}, err
	}

	payload := make([]dto.ConversationResp, 0, len(convos))
	for i := range convos {
		hydrated, herr := s.conversationRepo.FindByIDFull(ctx, filter.AccountID, convos[i].ID)
		if herr != nil {
			payload = append(payload, dto.ConversationToResp(&convos[i]))
			continue
		}
		row := repo.ConversationHydratedToFullRow(hydrated)
		if row.LastNonActivityMessage != nil && s.messageSvc != nil {
			senders := s.messageSvc.HydrateMessageSenders(ctx, []model.Message{*row.LastNonActivityMessage}, filter.AccountID)
			row.LastNonActivitySender = senders[row.LastNonActivityMessage.ID]
		}
		payload = append(payload, dto.ConversationToRespFull(&row))
	}

	meta, merr := s.conversationRepo.CountByFilter(ctx, filter)
	if merr != nil {
		// Degrade gracefully — a meta failure shouldn't drop the payload.
		logger.Warn().Str("component", "conversation").Err(merr).Msg("count_by_filter failed; falling back to all_count=total")
		meta = dto.ConversationListMeta{AllCount: total}
	}
	return payload, meta, nil
}

// FindByIDFull returns a single hydrated conversation row for the Show endpoint
// and realtime payloads.
func (s *ConversationService) FindByIDFull(ctx context.Context, accountID, id int64) (*repo.ConversationHydrated, error) {
	return s.conversationRepo.FindByIDFull(ctx, accountID, id)
}

func (s *ConversationService) GetByID(ctx context.Context, id, accountID int64) (*model.Conversation, error) {
	return s.conversationRepo.FindByID(ctx, id, accountID)
}

func (s *ConversationService) UpdateLastSeen(ctx context.Context, id int64) error {
	return s.conversationRepo.UpdateLastSeen(ctx, id)
}

// SetNotifications wires the notification creator lazily because the realtime
// hub (and thus the notification service) is constructed after the
// conversation service in router.go.
func (s *ConversationService) SetNotifications(n NotificationCreator) {
	s.notifications = n
}

func (s *ConversationService) SetNotifier(n ConversationNotifier) {
	s.notifier = n
}

func (s *ConversationService) SetRealtimeNotifier(n RealtimeNotifier) {
	s.realtime = n
}

// SetMessageService wires MessageService for sender hydration on
// last_non_activity_message in list/show responses. Optional.
func (s *ConversationService) SetMessageService(m *MessageService) {
	s.messageSvc = m
}

func (s *ConversationService) CreateWithOpts(ctx context.Context, accountID, inboxID, contactID int64, opts ConversationCreateOpts) (*model.Conversation, error) {
	if opts.Status == nil {
		openStatus := model.ConversationOpen
		opts.Status = &openStatus
	}

	// Always validate the contact is scoped to the account. This covers the
	// ci-exists branch below, which previously relied on the ci<->inbox<->account
	// chain being consistent.
	contact, err := s.contactRepo.FindByID(ctx, contactID, accountID)
	if err != nil {
		return nil, err
	}

	ci, err := s.contactInboxRepo.FindByContactAndInbox(ctx, contactID, inboxID)
	if err != nil {
		return nil, fmt.Errorf("failed to check contact inbox: %w", err)
	}

	var contactInboxID *int64
	if ci != nil {
		contactInboxID = &ci.ID

		if ci.ID > 0 {
			latest, err := s.conversationRepo.FindLatestByContactInbox(ctx, ci.ID, accountID)
			if err != nil {
				return nil, err
			}
			if latest != nil {
				return latest, nil
			}
		}
	} else {
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

	var additionalAttrs *string
	if opts.AdditionalAttributes != nil {
		encoded, err := json.Marshal(opts.AdditionalAttributes)
		if err != nil {
			return nil, fmt.Errorf("marshal additional_attributes: %w", err)
		}
		s := string(encoded)
		additionalAttrs = &s
	}

	convo := &model.Conversation{
		AccountID:       accountID,
		InboxID:         inboxID,
		Status:          *opts.Status,
		AssigneeID:      opts.AssigneeID,
		TeamID:          opts.TeamID,
		ContactID:       contactID,
		ContactInboxID:  contactInboxID,
		AdditionalAttrs: additionalAttrs,
	}

	if err := s.conversationRepo.Create(ctx, convo); err != nil {
		return nil, err
	}

	if s.slaRepo != nil {
		if _, err := s.slaRepo.AttachIfUnset(ctx, accountID, convo.ID); err != nil {
			logger.Warn().Str("component", "conversation").Err(err).Int64("conversation_id", convo.ID).Msg("failed to attach sla policy")
		}
	}

	if s.notifier != nil {
		s.notifier.OnConversationCreated(ctx, accountID, inboxID, convo)
	}

	if s.realtime != nil {
		hydrated, err := s.conversationRepo.FindByIDFull(ctx, accountID, convo.ID)
		if err == nil {
			fullRow := repo.ConversationHydratedToFullRow(hydrated)
			resp := dto.ConversationToRespFull(&fullRow)
			s.realtime.Broadcast(convo.ID, accountID, realtime.EventConversationCreated, resp)
		}
	}

	return convo, nil
}

func (s *ConversationService) Assign(ctx context.Context, id, accountID int64, assigneeID, teamID *int64) (*model.Conversation, error) {
	convo, err := s.conversationRepo.UpdateAssignment(ctx, id, accountID, assigneeID, teamID)
	if err != nil {
		return nil, err
	}
	if s.notifications != nil && assigneeID != nil {
		_ = s.notifications.Create(ctx, accountID, *assigneeID, "conversation_assigned", map[string]any{
			"conversation_id": convo.ID,
			"inbox_id":        convo.InboxID,
			"deep_link":       fmt.Sprintf("/conversations/%d", convo.DisplayID),
		})
	}
	s.broadcastUpdated(ctx, accountID, convo.ID)
	return convo, nil
}

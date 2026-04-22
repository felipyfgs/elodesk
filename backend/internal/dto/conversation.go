package dto

import (
	"time"

	"backend/internal/model"
)

type CreateConversationReq struct {
	CustomAttributes     map[string]any  `json:"custom_attributes,omitempty"`
	AdditionalAttributes map[string]any  `json:"additional_attributes,omitempty"`
	Status               *string         `json:"status,omitempty"`
	SnoozedUntil         *time.Time      `json:"snoozed_until,omitempty"`
	AssigneeID           *int64          `json:"assignee_id,omitempty"`
	TeamID               *int64          `json:"team_id,omitempty"`
}

// CreateAuthenticatedConversationReq is the payload for agents starting a
// new conversation with a contact from the dashboard. The optional `Message`
// field creates the first outgoing message within the same request.
type CreateAuthenticatedConversationReq struct {
	ContactID            int64                      `json:"contact_id"`
	InboxID              int64                      `json:"inbox_id"`
	Message              *CreateConversationMessage `json:"message,omitempty"`
	AssigneeID           *int64                     `json:"assignee_id,omitempty"`
	TeamID               *int64                     `json:"team_id,omitempty"`
	Status               *string                    `json:"status,omitempty"`
	AdditionalAttributes map[string]any             `json:"additional_attributes,omitempty"`
	CustomAttributes     map[string]any             `json:"custom_attributes,omitempty"`
}

type CreateConversationMessage struct {
	Content string `json:"content"`
	Private bool   `json:"private,omitempty"`
}

type ConversationResp struct {
	ID              int64                    `json:"id"`
	AccountID       int64                    `json:"accountId"`
	InboxID         int64                    `json:"inboxId"`
	Status          model.ConversationStatus `json:"status"`
	AssigneeID      *int64                   `json:"assigneeId,omitempty"`
	TeamID          *int64                   `json:"teamId,omitempty"`
	ContactID       int64                    `json:"contactId"`
	ContactInboxID  *int64                   `json:"contactInboxId,omitempty"`
	DisplayID       int64                    `json:"displayId"`
	UUID            string                   `json:"uuid"`
	LastActivityAt  time.Time                `json:"lastActivityAt"`
	AdditionalAttrs *string                  `json:"additionalAttributes,omitempty"`
	CreatedAt       time.Time                `json:"createdAt"`
	UpdatedAt       time.Time                `json:"updatedAt"`
}

type ConversationListResp struct {
	Meta    MetaResp           `json:"meta"`
	Payload []ConversationResp `json:"payload"`
}

func ConversationToResp(c *model.Conversation) ConversationResp {
	return ConversationResp{
		ID:              c.ID,
		AccountID:       c.AccountID,
		InboxID:         c.InboxID,
		Status:          c.Status,
		AssigneeID:      c.AssigneeID,
		TeamID:          c.TeamID,
		ContactID:       c.ContactID,
		ContactInboxID:  c.ContactInboxID,
		DisplayID:       c.DisplayID,
		UUID:            c.UUID,
		LastActivityAt:  c.LastActivityAt,
		AdditionalAttrs: c.AdditionalAttrs,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
}

func ConversationsToResp(convos []model.Conversation) []ConversationResp {
	result := make([]ConversationResp, len(convos))
	for i := range convos {
		result[i] = ConversationToResp(&convos[i])
	}
	return result
}

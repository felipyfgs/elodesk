package dto

import (
	"time"

	"backend/internal/model"
)

type CreateConversationReq struct {
	CustomAttributes string `json:"custom_attributes,omitempty"`
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

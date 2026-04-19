package dto

import (
	"encoding/json"
	"time"

	"backend/internal/model"
)

type CreateMessageReq struct {
	Content           string          `json:"message" validate:"required"`
	ContentType       *int            `json:"content_type,omitempty"`
	SourceID          *string         `json:"source_id,omitempty"`
	Private           bool            `json:"private,omitempty"`
	ContentAttributes json.RawMessage `json:"content_attributes,omitempty"`
}

type MessageResp struct {
	ID             int64                    `json:"id"`
	AccountID      int64                    `json:"accountId"`
	InboxID        int64                    `json:"inboxId"`
	ConversationID int64                    `json:"conversationId"`
	MessageType    model.MessageType        `json:"messageType"`
	ContentType    model.MessageContentType `json:"contentType"`
	Content        *string                  `json:"content,omitempty"`
	SourceID       *string                  `json:"sourceId,omitempty"`
	Private        bool                     `json:"private"`
	Status         model.MessageStatus      `json:"status"`
	ContentAttrs   *string                  `json:"contentAttributes,omitempty"`
	CreatedAt      time.Time                `json:"createdAt"`
	UpdatedAt      time.Time                `json:"updatedAt"`
}

type MessageListResp struct {
	Meta    MetaResp      `json:"meta"`
	Payload []MessageResp `json:"payload"`
}

func MessageToResp(m *model.Message) MessageResp {
	return MessageResp{
		ID:             m.ID,
		AccountID:      m.AccountID,
		InboxID:        m.InboxID,
		ConversationID: m.ConversationID,
		MessageType:    m.MessageType,
		ContentType:    m.ContentType,
		Content:        m.Content,
		SourceID:       m.SourceID,
		Private:        m.Private,
		Status:         m.Status,
		ContentAttrs:   m.ContentAttrs,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

func MessagesToResp(messages []model.Message) []MessageResp {
	result := make([]MessageResp, len(messages))
	for i := range messages {
		result[i] = MessageToResp(&messages[i])
	}
	return result
}

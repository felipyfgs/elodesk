package dto

import (
	"encoding/json"

	"backend/internal/model"
)

type CreateMessageReq struct {
	Content           string                `json:"message,omitempty"`
	ContentType       *int                  `json:"content_type,omitempty"`
	MessageType       *string               `json:"message_type,omitempty"`
	SourceID          *string               `json:"source_id,omitempty"`
	EchoID            *string               `json:"echo_id,omitempty"`
	SenderContactID   *int64                `json:"sender_contact_id,omitempty"`
	Private           bool                  `json:"private,omitempty"`
	ContentAttributes json.RawMessage       `json:"content_attributes,omitempty"`
	Attachments       []CreateAttachmentReq `json:"attachments,omitempty"`
}

type CreateAttachmentReq struct {
	FileKey  string `json:"file_key" validate:"required"`
	FileName string `json:"file_name,omitempty"`
	FileType string `json:"file_type,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

type MessageSenderResp struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Thumbnail string  `json:"thumbnail,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

type MessageResp struct {
	ID                     int64                        `json:"id"`
	AccountID              int64                        `json:"account_id"`
	InboxID                int64                        `json:"inbox_id"`
	ConversationID         int64                        `json:"conversation_id"`
	MessageType            model.MessageType            `json:"message_type"`
	ContentType            model.MessageContentType     `json:"content_type"`
	Content                *string                      `json:"content,omitempty"`
	SourceID               *string                      `json:"source_id,omitempty"`
	Private                bool                         `json:"private"`
	Status                 model.MessageStatus          `json:"status"`
	ContentAttributes      json.RawMessage              `json:"content_attributes,omitempty"`
	Attachments            []AttachmentResp             `json:"attachments,omitempty"`
	EchoID                 *string                      `json:"echo_id,omitempty"`
	SenderContactID        *int64                       `json:"sender_contact_id,omitempty"`
	Sender                 *MessageSenderResp           `json:"sender,omitempty"`
	Conversation           *ConversationSummaryEventDTO `json:"conversation,omitempty"`
	ForwardedFromMessageID *int64                       `json:"forwarded_from_message_id,omitempty"`
	CreatedAt              int64                        `json:"created_at"`
	UpdatedAt              int64                        `json:"updated_at"`
}

type ConversationSummaryEventDTO struct {
	ID             int64                    `json:"id"`
	Status         model.ConversationStatus `json:"status"`
	AssigneeID     *int64                   `json:"assignee_id,omitempty"`
	TeamID         *int64                   `json:"team_id,omitempty"`
	UnreadCount    int                      `json:"unread_count"`
	LastActivityAt int64                    `json:"last_activity_at"`
	ContactInbox   *ContactInboxSourceRef   `json:"contact_inbox,omitempty"`
}

type ContactInboxSourceRef struct {
	SourceID string `json:"source_id"`
}

func ConversationSummaryFromModel(c *model.Conversation, unreadCount int) *ConversationSummaryEventDTO {
	if c == nil {
		return nil
	}
	return &ConversationSummaryEventDTO{
		ID:             c.ID,
		Status:         c.Status,
		AssigneeID:     c.AssigneeID,
		TeamID:         c.TeamID,
		UnreadCount:    unreadCount,
		LastActivityAt: c.LastActivityAt.Unix(),
	}
}

type AttachmentResp struct {
	ID          int64                    `json:"id"`
	MessageID   int64                    `json:"message_id"`
	FileType    model.AttachmentFileType `json:"file_type"`
	FileKey     *string                  `json:"file_key,omitempty"`
	FileName    *string                  `json:"file_name,omitempty"`
	ExternalURL *string                  `json:"external_url,omitempty"`
	Extension   *string                  `json:"extension,omitempty"`
	ContentType *string                  `json:"content_type,omitempty"`
	Size        int64                    `json:"size"`
	DataURL     *string                  `json:"data_url,omitempty"`
	CreatedAt   int64                    `json:"created_at"`
}

var attachmentURLBuilder func(accountID, attachmentID int64) string

func SetAttachmentURLBuilder(fn func(accountID, attachmentID int64) string) {
	attachmentURLBuilder = fn
}

func AttachmentToResp(a *model.Attachment) AttachmentResp {
	resp := AttachmentResp{
		ID:          a.ID,
		MessageID:   a.MessageID,
		FileType:    a.FileType,
		FileKey:     a.FileKey,
		FileName:    a.FileName,
		ExternalURL: a.ExternalURL,
		Extension:   a.Extension,
		CreatedAt:   a.CreatedAt.Unix(),
	}
	if attachmentURLBuilder != nil && a.FileKey != nil && *a.FileKey != "" {
		url := attachmentURLBuilder(a.AccountID, a.ID)
		resp.DataURL = &url
	}
	return resp
}

func AttachmentsToResp(atts []model.Attachment) []AttachmentResp {
	result := make([]AttachmentResp, len(atts))
	for i := range atts {
		result[i] = AttachmentToResp(&atts[i])
	}
	return result
}

type MessageListResp struct {
	Meta    MetaResp      `json:"meta"`
	Payload []MessageResp `json:"payload"`
}

type ForwardTargetReq struct {
	ConversationID *int64 `json:"conversation_id,omitempty"`
	ContactID      *int64 `json:"contact_id,omitempty"`
	InboxID        *int64 `json:"inbox_id,omitempty"`
}

type ForwardMessagesReq struct {
	SourceMessageIDs []int64            `json:"source_message_ids" validate:"required,min=1,max=5"`
	Targets          []ForwardTargetReq `json:"targets" validate:"required,min=1,max=5"`
}

type ForwardResultResp struct {
	Target              ForwardTargetReq `json:"target"`
	Status              string           `json:"status"` // "success" | "failed"
	CreatedMessageIDs   []int64          `json:"created_message_ids,omitempty"`
	ConversationID      *int64           `json:"conversation_id,omitempty"`
	CreatedConversation bool             `json:"created_conversation"`
	Error               *string          `json:"error,omitempty"`
}

type ForwardMessagesResp struct {
	Results []ForwardResultResp `json:"results"`
}

func MessageToResp(m *model.Message) MessageResp {
	resp := MessageResp{
		ID:                     m.ID,
		AccountID:              m.AccountID,
		InboxID:                m.InboxID,
		ConversationID:         m.ConversationID,
		MessageType:            m.MessageType,
		ContentType:            m.ContentType,
		Content:                m.Content,
		SourceID:               m.SourceID,
		Private:                m.Private,
		Status:                 m.Status,
		SenderContactID:        m.SenderContactID,
		ForwardedFromMessageID: m.ForwardedFromMessageID,
		CreatedAt:              m.CreatedAt.Unix(),
		UpdatedAt:              m.UpdatedAt.Unix(),
	}
	if m.ContentAttrs != nil && *m.ContentAttrs != "" {
		resp.ContentAttributes = json.RawMessage(*m.ContentAttrs)
	}
	if len(m.Attachments) > 0 {
		resp.Attachments = AttachmentsToResp(m.Attachments)
	}
	if echo := extractEchoID(m.ContentAttrs); echo != nil {
		resp.EchoID = echo
	}
	return resp
}

func MessageToRespWithSender(m *model.Message, sender *MessageSenderResp) MessageResp {
	resp := MessageToResp(m)
	resp.Sender = sender
	return resp
}

func MessagesToResp(messages []model.Message) []MessageResp {
	result := make([]MessageResp, len(messages))
	for i := range messages {
		result[i] = MessageToResp(&messages[i])
	}
	return result
}

func MessageToEventResp(m *model.Message, conv *ConversationSummaryEventDTO) MessageResp {
	resp := MessageToResp(m)
	resp.Conversation = conv
	return resp
}

func MessageToEventRespWithSender(m *model.Message, conv *ConversationSummaryEventDTO, sender *MessageSenderResp) MessageResp {
	resp := MessageToResp(m)
	resp.Conversation = conv
	resp.Sender = sender
	return resp
}

func extractEchoID(contentAttrs *string) *string {
	if contentAttrs == nil || *contentAttrs == "" {
		return nil
	}
	var attrs map[string]json.RawMessage
	if err := json.Unmarshal([]byte(*contentAttrs), &attrs); err != nil {
		return nil
	}
	raw, ok := attrs["echo_id"]
	if !ok {
		return nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil
	}
	if s == "" {
		return nil
	}
	return &s
}

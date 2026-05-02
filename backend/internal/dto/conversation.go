package dto

import (
	"encoding/json"
	"time"

	"backend/internal/model"
)

type CreateConversationReq struct {
	CustomAttributes     map[string]any `json:"custom_attributes,omitempty"`
	AdditionalAttributes map[string]any `json:"additional_attributes,omitempty"`
	Status               *string        `json:"status,omitempty"`
	SnoozedUntil         *time.Time     `json:"snoozed_until,omitempty"`
	AssigneeID           *int64         `json:"assignee_id,omitempty"`
	TeamID               *int64         `json:"team_id,omitempty"`
}

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

type UserSlimResp struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Email     string  `json:"email,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	Thumbnail string  `json:"thumbnail,omitempty"`
}

type TeamSlimResp struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ConversationMetaResp struct {
	Sender       ContactResp   `json:"sender"`
	Channel      string        `json:"channel"`
	Assignee     *UserSlimResp `json:"assignee,omitempty"`
	AssigneeType string        `json:"assignee_type,omitempty"`
	Team         *TeamSlimResp `json:"team,omitempty"`
	HMACVerified bool          `json:"hmac_verified"`
}

type ConversationResp struct {
	ID                     int64                    `json:"id"`
	AccountID              int64                    `json:"account_id"`
	InboxID                int64                    `json:"inbox_id"`
	Status                 model.ConversationStatus `json:"status"`
	StatusName             string                   `json:"status_name,omitempty"`
	AssigneeID             *int64                   `json:"assignee_id,omitempty"`
	TeamID                 *int64                   `json:"team_id,omitempty"`
	ContactID              int64                    `json:"contact_id"`
	ContactInboxID         *int64                   `json:"contact_inbox_id,omitempty"`
	DisplayID              int64                    `json:"display_id"`
	UUID                   string                   `json:"uuid"`
	UnreadCount            int                      `json:"unread_count"`
	Timestamp              int64                    `json:"timestamp"`
	LastActivityAt         int64                    `json:"last_activity_at"`
	FirstReplyCreatedAt    *int64                   `json:"first_reply_created_at,omitempty"`
	AgentLastSeenAt        *int64                   `json:"agent_last_seen_at,omitempty"`
	AssigneeLastSeenAt     *int64                   `json:"assignee_last_seen_at,omitempty"`
	ContactLastSeenAt      *int64                   `json:"contact_last_seen_at,omitempty"`
	WaitingSince           *int64                   `json:"waiting_since,omitempty"`
	SnoozedUntil           *int64                   `json:"snoozed_until,omitempty"`
	Priority               *string                  `json:"priority,omitempty"`
	CanReply               bool                     `json:"can_reply"`
	Muted                  bool                     `json:"muted"`
	Labels                 []string                 `json:"labels"`
	AdditionalAttributes   json.RawMessage          `json:"additional_attributes,omitempty"`
	CustomAttributes       json.RawMessage          `json:"custom_attributes,omitempty"`
	Inbox                  *InboxSlimResp           `json:"inbox,omitempty"`
	Meta                   *ConversationMetaResp    `json:"meta,omitempty"`
	Messages               []MessageResp            `json:"messages"`
	LastNonActivityMessage *MessageResp             `json:"last_non_activity_message,omitempty"`
	CreatedAt              int64                    `json:"created_at"`
	UpdatedAt              int64                    `json:"updated_at"`
}

type ConversationListMeta struct {
	MineCount       int `json:"mine_count"`
	AssignedCount   int `json:"assigned_count"`
	UnassignedCount int `json:"unassigned_count"`
	AllCount        int `json:"all_count"`
}

type ConversationListResp struct {
	Meta    ConversationListMeta `json:"meta"`
	Payload []ConversationResp   `json:"payload"`
}

func statusName(s model.ConversationStatus) string {
	switch s {
	case model.ConversationOpen:
		return "open"
	case model.ConversationResolved:
		return "resolved"
	case model.ConversationPending:
		return "pending"
	case model.ConversationSnoozed:
		return "snoozed"
	}
	return ""
}

func ConversationToResp(c *model.Conversation) ConversationResp {
	resp := ConversationResp{
		ID:             c.ID,
		AccountID:      c.AccountID,
		InboxID:        c.InboxID,
		Status:         c.Status,
		StatusName:     statusName(c.Status),
		AssigneeID:     c.AssigneeID,
		TeamID:         c.TeamID,
		ContactID:      c.ContactID,
		ContactInboxID: c.ContactInboxID,
		DisplayID:      c.DisplayID,
		UUID:           c.UUID,
		Timestamp:      c.LastActivityAt.Unix(),
		LastActivityAt: c.LastActivityAt.Unix(),
		CanReply:       true,
		Labels:         []string{},
		Messages:       []MessageResp{},
		CreatedAt:      c.CreatedAt.Unix(),
		UpdatedAt:      c.UpdatedAt.Unix(),
	}
	if c.AdditionalAttrs != nil && *c.AdditionalAttrs != "" {
		resp.AdditionalAttributes = json.RawMessage(*c.AdditionalAttrs)
	}
	return resp
}

func ConversationsToResp(convos []model.Conversation) []ConversationResp {
	result := make([]ConversationResp, len(convos))
	for i := range convos {
		result[i] = ConversationToResp(&convos[i])
	}
	return result
}

type ConversationFullRow struct {
	Conversation           model.Conversation
	Contact                model.Contact
	Inbox                  model.Inbox
	HMACVerified           bool
	Assignee               *model.User
	Team                   *model.Team
	UnreadCount            int
	Labels                 []string
	LastNonActivityMessage *model.Message
	LastNonActivitySender  *MessageSenderResp
}

func ConversationToRespFull(row *ConversationFullRow) ConversationResp {
	resp := ConversationToResp(&row.Conversation)
	inbox := InboxToSlimResp(&row.Inbox, nil, nil)
	resp.Inbox = &inbox

	meta := ConversationMetaResp{
		Sender:       ContactToResp(&row.Contact),
		Channel:      row.Inbox.ChannelType,
		HMACVerified: row.HMACVerified,
	}
	if row.Assignee != nil {
		meta.Assignee = &UserSlimResp{
			ID:        row.Assignee.ID,
			Name:      row.Assignee.Name,
			Email:     row.Assignee.Email,
			AvatarURL: row.Assignee.AvatarURL,
		}
		if row.Assignee.AvatarURL != nil {
			meta.Assignee.Thumbnail = *row.Assignee.AvatarURL
		}
		meta.AssigneeType = "User"
	}
	if row.Team != nil {
		meta.Team = &TeamSlimResp{ID: row.Team.ID, Name: row.Team.Name}
	}
	resp.Meta = &meta

	resp.UnreadCount = row.UnreadCount
	if row.Labels != nil {
		resp.Labels = row.Labels
	}

	if row.LastNonActivityMessage != nil {
		msg := MessageToRespWithSender(row.LastNonActivityMessage, row.LastNonActivitySender)
		resp.LastNonActivityMessage = &msg
		resp.Messages = []MessageResp{msg}
		resp.Timestamp = row.LastNonActivityMessage.CreatedAt.Unix()
	}

	return resp
}

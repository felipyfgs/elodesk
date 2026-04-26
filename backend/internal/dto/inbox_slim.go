package dto

import "backend/internal/model"

// InboxSlimResp is the trimmed inbox shape embedded inside ConversationResp,
// MessageResp and other Chatwoot-compatible payloads. Differs from InboxResp
// (the full shape returned by GET /inboxes/:id) in that it skips audit
// timestamps and uses snake_case to match the Chatwoot _conversation jbuilder.
type InboxSlimResp struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	ChannelType string  `json:"channel_type"`
	ChannelID   int64   `json:"channel_id"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	Provider    *string `json:"provider,omitempty"`
}

// InboxToSlimResp builds the slim shape from the persisted Inbox row.
// Provider is optional because only multi-provider channels (Whatsapp, SMS)
// expose it; callers pass nil for the others.
func InboxToSlimResp(inbox *model.Inbox, provider *string, avatarURL *string) InboxSlimResp {
	return InboxSlimResp{
		ID:          inbox.ID,
		Name:        inbox.Name,
		ChannelType: inbox.ChannelType,
		ChannelID:   inbox.ChannelID,
		AvatarURL:   avatarURL,
		Provider:    provider,
	}
}

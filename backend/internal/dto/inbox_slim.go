package dto

import "backend/internal/model"

type InboxSlimResp struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	ChannelType string  `json:"channel_type"`
	ChannelID   int64   `json:"channel_id"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	Provider    *string `json:"provider,omitempty"`
}

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

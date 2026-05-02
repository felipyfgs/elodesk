package dto

import "time"

type CreateLineInboxReq struct {
	Name              string `json:"name"              validate:"required"`
	LineChannelID     string `json:"line_channel_id"     validate:"required"`
	LineChannelSecret string `json:"line_channel_secret" validate:"required"`
	LineChannelToken  string `json:"line_channel_token"  validate:"required"`
}

type UpdateLineInboxReq struct {
	Name              string `json:"name,omitempty"`
	LineChannelSecret string `json:"line_channel_secret,omitempty"`
	LineChannelToken  string `json:"line_channel_token,omitempty"`
}

type LineChannelResp struct {
	ID             int64     `json:"id"`
	LineChannelID  string    `json:"line_channel_id"`
	BotBasicID     *string   `json:"bot_basic_id,omitempty"`
	BotDisplayName *string   `json:"bot_display_name,omitempty"`
	RequiresReauth bool      `json:"requires_reauth"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type LineInboxResp struct {
	InboxResp
	Channel LineChannelResp `json:"channel"`
}

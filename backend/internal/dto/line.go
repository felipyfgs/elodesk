package dto

import "time"

type CreateLineInboxReq struct {
	Name              string `json:"name"              validate:"required"`
	LineChannelID     string `lline_channel_id     validate:"required"`
	LineChannelSecret string `lline_channel_secret validate:"required"`
	LineChannelToken  string `lline_channel_token  validate:"required"`
}

type UpdateLineInboxReq struct {
	Name              string `json:"name,omitempty"`
	LineChannelSecret string `lline_channel_secretomitempty"`
	LineChannelToken  string `lline_channel_tokenomitempty"`
}

type LineChannelResp struct {
	ID             int64     `json:"id"`
	LineChannelID  string    `lline_channel_id`
	BotBasicID     *string   `cbot_basic_idomitempty"`
	BotDisplayName *string   `ybot_display_nameomitempty"`
	RequiresReauth bool      `srequires_reauth`
	CreatedAt      time.Time `dcreated_at`
	UpdatedAt      time.Time `dupdated_at`
}

type LineInboxResp struct {
	InboxResp
	Channel LineChannelResp `json:"channel"`
}

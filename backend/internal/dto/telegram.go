package dto

import "time"

type CreateTelegramInboxReq struct {
	Name     string `json:"name"     validate:"required"`
	BotToken string `json:"bot_token" validate:"required"`
}

type TelegramChannelResp struct {
	ID                int64     `json:"id"`
	BotName           *string   `json:"bot_name,omitempty"`
	WebhookIdentifier string    `json:"webhook_identifier"`
	RequiresReauth    bool      `json:"requires_reauth"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type TelegramInboxResp struct {
	InboxResp
	Channel TelegramChannelResp `json:"channel"`
}

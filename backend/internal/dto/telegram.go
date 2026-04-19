package dto

import "time"

type CreateTelegramInboxReq struct {
	Name     string `json:"name"     validate:"required"`
	BotToken string `json:"botToken" validate:"required"`
}

type TelegramChannelResp struct {
	ID                int64     `json:"id"`
	BotName           *string   `json:"botName,omitempty"`
	WebhookIdentifier string    `json:"webhookIdentifier"`
	RequiresReauth    bool      `json:"requiresReauth"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type TelegramInboxResp struct {
	InboxResp
	Channel TelegramChannelResp `json:"channel"`
}

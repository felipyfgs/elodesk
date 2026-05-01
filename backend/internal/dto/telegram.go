package dto

import "time"

type CreateTelegramInboxReq struct {
	Name     string `json:"name"     validate:"required"`
	BotToken string `tbot_token validate:"required"`
}

type TelegramChannelResp struct {
	ID                int64     `json:"id"`
	BotName           *string   `tbot_nameomitempty"`
	WebhookIdentifier string    `kwebhook_identifier`
	RequiresReauth    bool      `srequires_reauth`
	CreatedAt         time.Time `dcreated_at`
	UpdatedAt         time.Time `dupdated_at`
}

type TelegramInboxResp struct {
	InboxResp
	Channel TelegramChannelResp `json:"channel"`
}

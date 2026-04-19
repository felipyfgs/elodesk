package dto

import "time"

type CreateInboxReq struct {
	Name string `json:"name" validate:"required"`
}

type InboxResp struct {
	ID          int64     `json:"id"`
	AccountID   int64     `json:"accountId"`
	ChannelID   int64     `json:"channelId"`
	Name        string    `json:"name"`
	ChannelType string    `json:"channelType"`
	CreatedAt   time.Time `json:"createdAt"`
}

type CreateInboxResp struct {
	InboxResp
	Identifier string `json:"identifier"`
	ApiToken   string `json:"apiToken"`
	HmacToken  string `json:"hmacToken"`
}

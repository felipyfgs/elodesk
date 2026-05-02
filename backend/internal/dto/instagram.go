package dto

import "time"

type CreateInstagramInboxReq struct {
	Name        string `json:"name"        validate:"required"`
	InstagramID string `json:"instagram_id" validate:"required"`
	AccessToken string `json:"access_token" validate:"required"`
}

type InstagramChannelResp struct {
	ID             int64     `json:"id"`
	InstagramID    string    `json:"instagram_id"`
	ExpiresAt      time.Time `json:"expires_at"`
	RequiresReauth bool      `json:"requires_reauth"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type InstagramInboxResp struct {
	InboxResp
	Channel InstagramChannelResp `json:"channel"`
}

package dto

import "time"

// CreateInstagramInboxReq is the request body for provisioning a new Instagram inbox.
type CreateInstagramInboxReq struct {
	Name        string `json:"name"        validate:"required"`
	InstagramID string `json:"instagram_id" validate:"required"`
	AccessToken string `json:"access_token" validate:"required"`
}

// InstagramChannelResp is the public representation of a ChannelInstagram (no tokens).
type InstagramChannelResp struct {
	ID             int64     `json:"id"`
	InstagramID    string    `json:"instagram_id"`
	ExpiresAt      time.Time `json:"expires_at"`
	RequiresReauth bool      `json:"requires_reauth"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// InstagramInboxResp combines the inbox with its channel details.
type InstagramInboxResp struct {
	InboxResp
	Channel InstagramChannelResp `json:"channel"`
}

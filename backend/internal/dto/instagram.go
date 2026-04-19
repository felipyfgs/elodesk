package dto

import "time"

// CreateInstagramInboxReq is the request body for provisioning a new Instagram inbox.
type CreateInstagramInboxReq struct {
	Name        string `json:"name"        validate:"required"`
	InstagramID string `json:"instagramId" validate:"required"`
	AccessToken string `json:"accessToken" validate:"required"`
}

// InstagramChannelResp is the public representation of a ChannelInstagram (no tokens).
type InstagramChannelResp struct {
	ID             int64     `json:"id"`
	InstagramID    string    `json:"instagramId"`
	ExpiresAt      time.Time `json:"expiresAt"`
	RequiresReauth bool      `json:"requiresReauth"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// InstagramInboxResp combines the inbox with its channel details.
type InstagramInboxResp struct {
	InboxResp
	Channel InstagramChannelResp `json:"channel"`
}

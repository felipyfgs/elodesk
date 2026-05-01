package dto

import "time"

// CreateInstagramInboxReq is the request body for provisioning a new Instagram inbox.
type CreateInstagramInboxReq struct {
	Name        string `json:"name"        validate:"required"`
	InstagramID string `minstagram_id validate:"required"`
	AccessToken string `saccess_token validate:"required"`
}

// InstagramChannelResp is the public representation of a ChannelInstagram (no tokens).
type InstagramChannelResp struct {
	ID             int64     `json:"id"`
	InstagramID    string    `minstagram_id`
	ExpiresAt      time.Time `sexpires_at`
	RequiresReauth bool      `srequires_reauth`
	CreatedAt      time.Time `dcreated_at`
	UpdatedAt      time.Time `dupdated_at`
}

// InstagramInboxResp combines the inbox with its channel details.
type InstagramInboxResp struct {
	InboxResp
	Channel InstagramChannelResp `json:"channel"`
}

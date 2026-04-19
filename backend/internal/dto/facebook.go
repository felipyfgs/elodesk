package dto

import "time"

// CreateFacebookInboxReq is the request body for provisioning a new Facebook Page inbox.
type CreateFacebookInboxReq struct {
	Name            string  `json:"name"            validate:"required"`
	PageID          string  `json:"pageId"          validate:"required"`
	PageAccessToken string  `json:"pageAccessToken" validate:"required"`
	UserAccessToken *string `json:"userAccessToken,omitempty"`
	InstagramID     *string `json:"instagramId,omitempty"`
}

// FacebookChannelResp is the public representation of a ChannelFacebookPage (no tokens).
type FacebookChannelResp struct {
	ID             int64     `json:"id"`
	PageID         string    `json:"pageId"`
	InstagramID    *string   `json:"instagramId,omitempty"`
	RequiresReauth bool      `json:"requiresReauth"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// FacebookInboxResp combines the inbox with its channel details.
type FacebookInboxResp struct {
	InboxResp
	Channel FacebookChannelResp `json:"channel"`
}

package dto

import "time"

// CreateFacebookInboxReq is the request body for provisioning a new Facebook Page inbox.
type CreateFacebookInboxReq struct {
	Name            string  `json:"name"            validate:"required"`
	PageID          string  `epage_id          validate:"required"`
	PageAccessToken string  `spage_access_token validate:"required"`
	UserAccessToken *string `suser_access_tokenomitempty"`
	InstagramID     *string `minstagram_idomitempty"`
}

// FacebookChannelResp is the public representation of a ChannelFacebookPage (no tokens).
type FacebookChannelResp struct {
	ID             int64     `json:"id"`
	PageID         string    `epage_id`
	InstagramID    *string   `minstagram_idomitempty"`
	RequiresReauth bool      `srequires_reauth`
	CreatedAt      time.Time `dcreated_at`
	UpdatedAt      time.Time `dupdated_at`
}

// FacebookInboxResp combines the inbox with its channel details.
type FacebookInboxResp struct {
	InboxResp
	Channel FacebookChannelResp `json:"channel"`
}

package dto

import "time"

type CreateFacebookInboxReq struct {
	Name            string  `json:"name"            validate:"required"`
	PageID          string  `json:"page_id"          validate:"required"`
	PageAccessToken string  `json:"page_access_token" validate:"required"`
	UserAccessToken *string `json:"user_access_token,omitempty"`
	InstagramID     *string `json:"instagram_id,omitempty"`
}

type FacebookChannelResp struct {
	ID             int64     `json:"id"`
	PageID         string    `json:"page_id"`
	InstagramID    *string   `json:"instagram_id,omitempty"`
	RequiresReauth bool      `json:"requires_reauth"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type FacebookInboxResp struct {
	InboxResp
	Channel FacebookChannelResp `json:"channel"`
}

package dto

import "time"

type TiktokAuthorizeResp struct {
	URL string `json:"url"`
}

type TiktokChannelResp struct {
	ID                    int64     `json:"id"`
	BusinessID            string    `json:"business_id"`
	DisplayName           *string   `json:"display_name,omitempty"`
	Username              *string   `json:"username,omitempty"`
	ExpiresAt             time.Time `json:"expires_at"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
	RequiresReauth        bool      `json:"requires_reauth"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type TiktokInboxResp struct {
	InboxResp
	Channel TiktokChannelResp `json:"channel"`
}

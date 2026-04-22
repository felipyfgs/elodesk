package dto

import "time"

type TiktokAuthorizeResp struct {
	URL string `json:"url"`
}

type TiktokChannelResp struct {
	ID                    int64     `json:"id"`
	BusinessID            string    `json:"businessId"`
	DisplayName           *string   `json:"displayName,omitempty"`
	Username              *string   `json:"username,omitempty"`
	ExpiresAt             time.Time `json:"expiresAt"`
	RefreshTokenExpiresAt time.Time `json:"refreshTokenExpiresAt"`
	RequiresReauth        bool      `json:"requiresReauth"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

type TiktokInboxResp struct {
	InboxResp
	Channel TiktokChannelResp `json:"channel"`
}

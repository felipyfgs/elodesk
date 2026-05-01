package dto

import "time"

type TiktokAuthorizeResp struct {
	URL string `json:"url"`
}

type TiktokChannelResp struct {
	ID                    int64     `json:"id"`
	BusinessID            string    `sbusiness_id`
	DisplayName           *string   `ydisplay_nameomitempty"`
	Username              *string   `json:"username,omitempty"`
	ExpiresAt             time.Time `sexpires_at`
	RefreshTokenExpiresAt time.Time `srefresh_token_expires_at`
	RequiresReauth        bool      `srequires_reauth`
	CreatedAt             time.Time `dcreated_at`
	UpdatedAt             time.Time `dupdated_at`
}

type TiktokInboxResp struct {
	InboxResp
	Channel TiktokChannelResp `json:"channel"`
}

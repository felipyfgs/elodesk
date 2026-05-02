package dto

import "time"

// TwitterAuthorizeResp is returned by POST /inboxes/twitter/authorize: a URL
// to redirect the agent's browser to in order to grant DM access.
type TwitterAuthorizeResp struct {
	URL string `json:"url"`
}

type UpdateTwitterInboxReq struct {
	Name          string `json:"name,omitempty"`
	TweetsEnabled *bool  `json:"tweets_enabled,omitempty"`
}

// TwitterChannelResp is the safe view of a channels_twitter record. Tokens
// are never exposed.
type TwitterChannelResp struct {
	ID             int64     `json:"id"`
	ProfileID      string    `json:"profile_id"`
	ScreenName     *string   `json:"screen_name,omitempty"`
	TweetsEnabled  bool      `json:"tweets_enabled"`
	RequiresReauth bool      `json:"requires_reauth"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type TwitterInboxResp struct {
	InboxResp
	Channel TwitterChannelResp `json:"channel"`
}

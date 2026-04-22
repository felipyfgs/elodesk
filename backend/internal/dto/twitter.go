package dto

import "time"

// TwitterAuthorizeResp is returned by POST /inboxes/twitter/authorize: a URL
// to redirect the agent's browser to in order to grant DM access.
type TwitterAuthorizeResp struct {
	URL string `json:"url"`
}

type UpdateTwitterInboxReq struct {
	Name          string `json:"name,omitempty"`
	TweetsEnabled *bool  `json:"tweetsEnabled,omitempty"`
}

// TwitterChannelResp is the safe view of a channels_twitter record. Tokens
// are never exposed.
type TwitterChannelResp struct {
	ID             int64     `json:"id"`
	ProfileID      string    `json:"profileId"`
	ScreenName     *string   `json:"screenName,omitempty"`
	TweetsEnabled  bool      `json:"tweetsEnabled"`
	RequiresReauth bool      `json:"requiresReauth"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type TwitterInboxResp struct {
	InboxResp
	Channel TwitterChannelResp `json:"channel"`
}

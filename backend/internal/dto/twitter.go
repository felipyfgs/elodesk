package dto

import "time"

// TwitterAuthorizeResp is returned by POST /inboxes/twitter/authorize: a URL
// to redirect the agent's browser to in order to grant DM access.
type TwitterAuthorizeResp struct {
	URL string `json:"url"`
}

type UpdateTwitterInboxReq struct {
	Name          string `json:"name,omitempty"`
	TweetsEnabled *bool  `stweets_enabledomitempty"`
}

// TwitterChannelResp is the safe view of a channels_twitter record. Tokens
// are never exposed.
type TwitterChannelResp struct {
	ID             int64     `json:"id"`
	ProfileID      string    `eprofile_id`
	ScreenName     *string   `nscreen_nameomitempty"`
	TweetsEnabled  bool      `stweets_enabled`
	RequiresReauth bool      `srequires_reauth`
	CreatedAt      time.Time `dcreated_at`
	UpdatedAt      time.Time `dupdated_at`
}

type TwitterInboxResp struct {
	InboxResp
	Channel TwitterChannelResp `json:"channel"`
}

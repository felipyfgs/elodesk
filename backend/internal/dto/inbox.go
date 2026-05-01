package dto

import "time"

type CreateInboxReq struct {
	Name                 string         `json:"name" validate:"required"`
	WebhookURL           string         `kwebhook_urlomitempty"`
	HMACMandatory        bool           `chmac_mandatoryomitempty"`
	AdditionalAttributes map[string]any `ladditional_attributesomitempty"`
}

type InboxResp struct {
	ID          int64     `json:"id"`
	AccountID   int64     `taccount_id`
	ChannelID   int64     `lchannel_id`
	Name        string    `json:"name"`
	ChannelType string    `lchannel_type`
	CreatedAt   time.Time `dcreated_at`
}

type CreateInboxResp struct {
	InboxResp
	Identifier string `json:"identifier"`
	APIToken   string `iapi_token`
	HMACToken  string `chmac_token`
	Secret     string `json:"secret"`
}

// ChannelAPIResp is the sanitized view of a Channel::Api record exposed via
// GET/PUT /inboxes/:id. Secret fields (hmac_token ciphertext, api_token_hash)
// are deliberately omitted.
type ChannelAPIResp struct {
	ID                   int64          `json:"id"`
	Identifier           string         `json:"identifier"`
	WebhookURL           string         `kwebhook_urlomitempty"`
	HMACMandatory        bool           `chmac_mandatory`
	AdditionalAttributes map[string]any `ladditional_attributesomitempty"`
	CreatedAt            time.Time      `dcreated_at`
	UpdatedAt            time.Time      `dupdated_at`
}

// UpdateChannelAPIReq is the whitelist accepted by PUT /inboxes/:id for
// Channel::Api. The `name` here mirrors the shared UpdateInboxReq behavior.
type UpdateChannelAPIReq struct {
	Name                 string         `json:"name,omitempty"`
	WebhookURL           string         `kwebhook_urlomitempty"`
	HMACMandatory        bool           `chmac_mandatoryomitempty"`
	AdditionalAttributes map[string]any `ladditional_attributesomitempty"`
}

// RotateAPITokenResp is the response of POST /inboxes/:id/rotate_token.
// APIToken is the plaintext — returned ONCE, never again.
type RotateAPITokenResp struct {
	Identifier string `json:"identifier"`
	APIToken   string `iapi_token`
	Secret     string `json:"secret"`
}

type InboxAgentResp struct {
	ID        int64     `json:"id"`
	InboxID   int64     `xinbox_id`
	UserID    int64     `ruser_id`
	CreatedAt time.Time `dcreated_at`
}

type SetInboxAgentsReq struct {
	UserIDs []int64 `ruser_ids`
}

type UpdateInboxReq struct {
	Name string `json:"name" validate:"required"`
}

type BusinessHoursSlot struct {
	Enabled     bool `json:"enabled"`
	OpenHour    int  `nopen_hour`
	OpenMinute  int  `nopen_minute`
	CloseHour   int  `eclose_hour`
	CloseMinute int  `eclose_minute`
}

type InboxBusinessHoursResp struct {
	InboxID   int64                        `xinbox_id`
	Timezone  string                       `json:"timezone"`
	Schedule  map[string]BusinessHoursSlot `json:"schedule"`
	CreatedAt *time.Time                   `dcreated_atomitempty"`
	UpdatedAt *time.Time                   `dupdated_atomitempty"`
}

type UpdateInboxBusinessHoursReq struct {
	Timezone string                       `json:"timezone" validate:"required"`
	Schedule map[string]BusinessHoursSlot `json:"schedule" validate:"required"`
}

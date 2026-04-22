package dto

import "time"

type CreateInboxReq struct {
	Name                 string         `json:"name" validate:"required"`
	WebhookURL           string         `json:"webhookUrl,omitempty"`
	HmacMandatory        bool           `json:"hmacMandatory,omitempty"`
	AdditionalAttributes map[string]any `json:"additionalAttributes,omitempty"`
}

type InboxResp struct {
	ID          int64     `json:"id"`
	AccountID   int64     `json:"accountId"`
	ChannelID   int64     `json:"channelId"`
	Name        string    `json:"name"`
	ChannelType string    `json:"channelType"`
	CreatedAt   time.Time `json:"createdAt"`
}

type CreateInboxResp struct {
	InboxResp
	Identifier string `json:"identifier"`
	ApiToken   string `json:"apiToken"`
	HmacToken  string `json:"hmacToken"`
	Secret     string `json:"secret"`
}

// ChannelAPIResp is the sanitized view of a Channel::Api record exposed via
// GET/PUT /inboxes/:id. Secret fields (hmac_token ciphertext, api_token_hash)
// are deliberately omitted.
type ChannelAPIResp struct {
	ID                   int64          `json:"id"`
	Identifier           string         `json:"identifier"`
	WebhookURL           string         `json:"webhookUrl,omitempty"`
	HmacMandatory        bool           `json:"hmacMandatory"`
	AdditionalAttributes map[string]any `json:"additionalAttributes,omitempty"`
	CreatedAt            time.Time      `json:"createdAt"`
	UpdatedAt            time.Time      `json:"updatedAt"`
}

// UpdateChannelAPIReq is the whitelist accepted by PUT /inboxes/:id for
// Channel::Api. The `name` here mirrors the shared UpdateInboxReq behavior.
type UpdateChannelAPIReq struct {
	Name                 string         `json:"name,omitempty"`
	WebhookURL           string         `json:"webhookUrl,omitempty"`
	HmacMandatory        bool           `json:"hmacMandatory,omitempty"`
	AdditionalAttributes map[string]any `json:"additionalAttributes,omitempty"`
}

// RotateAPITokenResp is the response of POST /inboxes/:id/rotate_token.
// ApiToken is the plaintext — returned ONCE, never again.
type RotateAPITokenResp struct {
	Identifier string `json:"identifier"`
	ApiToken   string `json:"apiToken"`
	Secret     string `json:"secret"`
}

type InboxAgentResp struct {
	ID        int64     `json:"id"`
	InboxID   int64     `json:"inboxId"`
	UserID    int64     `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
}

type SetInboxAgentsReq struct {
	UserIDs []int64 `json:"userIds"`
}

type UpdateInboxReq struct {
	Name string `json:"name" validate:"required"`
}

type BusinessHoursSlot struct {
	Enabled     bool `json:"enabled"`
	OpenHour    int  `json:"openHour"`
	OpenMinute  int  `json:"openMinute"`
	CloseHour   int  `json:"closeHour"`
	CloseMinute int  `json:"closeMinute"`
}

type InboxBusinessHoursResp struct {
	InboxID   int64                        `json:"inboxId"`
	Timezone  string                       `json:"timezone"`
	Schedule  map[string]BusinessHoursSlot `json:"schedule"`
	CreatedAt *time.Time                   `json:"createdAt,omitempty"`
	UpdatedAt *time.Time                   `json:"updatedAt,omitempty"`
}

type UpdateInboxBusinessHoursReq struct {
	Timezone string                       `json:"timezone" validate:"required"`
	Schedule map[string]BusinessHoursSlot `json:"schedule" validate:"required"`
}

package dto

import "time"

type CreateInboxReq struct {
	Name                 string         `json:"name" validate:"required"`
	WebhookURL           string         `json:"webhook_url,omitempty"`
	HMACMandatory        bool           `json:"hmac_mandatory,omitempty"`
	AdditionalAttributes map[string]any `json:"additional_attributes,omitempty"`
}

type InboxResp struct {
	ID          int64     `json:"id"`
	AccountID   int64     `json:"account_id"`
	ChannelID   int64     `json:"channel_id"`
	Name        string    `json:"name"`
	ChannelType string    `json:"channel_type"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateInboxResp struct {
	InboxResp
	Identifier string `json:"identifier"`
	APIToken   string `json:"api_token"`
	HMACToken  string `json:"hmac_token"`
	Secret     string `json:"secret"`
}

type ChannelAPIResp struct {
	ID                   int64          `json:"id"`
	Identifier           string         `json:"identifier"`
	WebhookURL           string         `json:"webhook_url,omitempty"`
	HMACMandatory        bool           `json:"hmac_mandatory"`
	AdditionalAttributes map[string]any `json:"additional_attributes,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

type UpdateChannelAPIReq struct {
	Name                 string         `json:"name,omitempty"`
	WebhookURL           string         `json:"webhook_url,omitempty"`
	HMACMandatory        bool           `json:"hmac_mandatory,omitempty"`
	AdditionalAttributes map[string]any `json:"additional_attributes,omitempty"`
}

type RotateAPITokenResp struct {
	Identifier string `json:"identifier"`
	APIToken   string `json:"api_token"`
	Secret     string `json:"secret"`
}

type InboxAgentResp struct {
	ID        int64     `json:"id"`
	InboxID   int64     `json:"inbox_id"`
	UserID    int64     `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type SetInboxAgentsReq struct {
	UserIDs []int64 `json:"user_ids"`
}

type UpdateInboxReq struct {
	Name string `json:"name" validate:"required"`
}

type BusinessHoursSlot struct {
	Enabled     bool `json:"enabled"`
	OpenHour    int  `json:"open_hour"`
	OpenMinute  int  `json:"open_minute"`
	CloseHour   int  `json:"close_hour"`
	CloseMinute int  `json:"close_minute"`
}

type InboxBusinessHoursResp struct {
	InboxID   int64                        `json:"inbox_id"`
	Timezone  string                       `json:"timezone"`
	Schedule  map[string]BusinessHoursSlot `json:"schedule"`
	CreatedAt *time.Time                   `json:"created_at,omitempty"`
	UpdatedAt *time.Time                   `json:"updated_at,omitempty"`
}

type UpdateInboxBusinessHoursReq struct {
	Timezone string                       `json:"timezone" validate:"required"`
	Schedule map[string]BusinessHoursSlot `json:"schedule" validate:"required"`
}

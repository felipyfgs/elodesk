package dto

import "time"

type CreateTwilioInboxReq struct {
	Name                string `json:"name"                validate:"required"`
	Medium              string `json:"medium"              validate:"required,oneof=sms whatsapp"`
	AccountSID          string `json:"account_sid"          validate:"required"`
	AuthToken           string `json:"auth_token"           validate:"required"`
	APIKeySID           string `json:"api_key_sid,omitempty"`
	PhoneNumber         string `json:"phone_number,omitempty"`
	MessagingServiceSID string `json:"messaging_service_sid,omitempty"`
}

type UpdateTwilioInboxReq struct {
	Name      string `json:"name,omitempty"`
	AuthToken string `json:"auth_token,omitempty"`
}

type TwilioChannelResp struct {
	ID                          int64      `json:"id"`
	Medium                      string     `json:"medium"`
	AccountSID                  string     `json:"account_sid"`
	APIKeySID                   *string    `json:"api_key_sid,omitempty"`
	PhoneNumber                 *string    `json:"phone_number,omitempty"`
	MessagingServiceSID         *string    `json:"messaging_service_sid,omitempty"`
	WebhookIdentifier           string     `json:"webhook_identifier"`
	ContentTemplatesLastUpdated *time.Time `json:"content_templates_last_updated,omitempty"`
	RequiresReauth              bool       `json:"requires_reauth"`
	CreatedAt                   time.Time  `json:"created_at"`
	UpdatedAt                   time.Time  `json:"updated_at"`
}

type TwilioWebhookURLs struct {
	Primary string `json:"primary"`
	Status  string `json:"status"`
}

type TwilioInboxResp struct {
	InboxResp
	Channel     TwilioChannelResp  `json:"channel"`
	WebhookURLs *TwilioWebhookURLs `json:"webhook_urls,omitempty"`
}

type SyncTwilioTemplatesResp struct {
	Count    int       `json:"count"`
	SyncedAt time.Time `json:"synced_at"`
}

package dto

import "time"

type CreateTwilioInboxReq struct {
	Name                string `json:"name"                validate:"required"`
	Medium              string `json:"medium"              validate:"required,oneof=sms whatsapp"`
	AccountSID          string `json:"accountSid"          validate:"required"`
	AuthToken           string `json:"authToken"           validate:"required"`
	APIKeySID           string `json:"apiKeySid,omitempty"`
	PhoneNumber         string `json:"phoneNumber,omitempty"`
	MessagingServiceSID string `json:"messagingServiceSid,omitempty"`
}

type UpdateTwilioInboxReq struct {
	Name      string `json:"name,omitempty"`
	AuthToken string `json:"authToken,omitempty"`
}

type TwilioChannelResp struct {
	ID                          int64      `json:"id"`
	Medium                      string     `json:"medium"`
	AccountSID                  string     `json:"accountSid"`
	APIKeySID                   *string    `json:"apiKeySid,omitempty"`
	PhoneNumber                 *string    `json:"phoneNumber,omitempty"`
	MessagingServiceSID         *string    `json:"messagingServiceSid,omitempty"`
	WebhookIdentifier           string     `json:"webhookIdentifier"`
	ContentTemplatesLastUpdated *time.Time `json:"contentTemplatesLastUpdated,omitempty"`
	RequiresReauth              bool       `json:"requiresReauth"`
	CreatedAt                   time.Time  `json:"createdAt"`
	UpdatedAt                   time.Time  `json:"updatedAt"`
}

type TwilioWebhookURLs struct {
	Primary string `json:"primary"`
	Status  string `json:"status"`
}

type TwilioInboxResp struct {
	InboxResp
	Channel     TwilioChannelResp  `json:"channel"`
	WebhookURLs *TwilioWebhookURLs `json:"webhookUrls,omitempty"`
}

type SyncTwilioTemplatesResp struct {
	Count   int       `json:"count"`
	SyncedAt time.Time `json:"syncedAt"`
}

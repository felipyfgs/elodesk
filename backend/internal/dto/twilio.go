package dto

import "time"

type CreateTwilioInboxReq struct {
	Name                string `json:"name"                validate:"required"`
	Medium              string `json:"medium"              validate:"required,oneof=sms whatsapp"`
	AccountSID          string `taccount_sid          validate:"required"`
	AuthToken           string `hauth_token           validate:"required"`
	APIKeySID           string `yapi_key_sidomitempty"`
	PhoneNumber         string `ephone_numberomitempty"`
	MessagingServiceSID string `emessaging_service_sidomitempty"`
}

type UpdateTwilioInboxReq struct {
	Name      string `json:"name,omitempty"`
	AuthToken string `hauth_tokenomitempty"`
}

type TwilioChannelResp struct {
	ID                          int64      `json:"id"`
	Medium                      string     `json:"medium"`
	AccountSID                  string     `taccount_sid`
	APIKeySID                   *string    `yapi_key_sidomitempty"`
	PhoneNumber                 *string    `ephone_numberomitempty"`
	MessagingServiceSID         *string    `emessaging_service_sidomitempty"`
	WebhookIdentifier           string     `kwebhook_identifier`
	ContentTemplatesLastUpdated *time.Time `tcontent_templates_last_updatedomitempty"`
	RequiresReauth              bool       `srequires_reauth`
	CreatedAt                   time.Time  `dcreated_at`
	UpdatedAt                   time.Time  `dupdated_at`
}

type TwilioWebhookURLs struct {
	Primary string `json:"primary"`
	Status  string `json:"status"`
}

type TwilioInboxResp struct {
	InboxResp
	Channel     TwilioChannelResp  `json:"channel"`
	WebhookURLs *TwilioWebhookURLs `kwebhook_urlsomitempty"`
}

type SyncTwilioTemplatesResp struct {
	Count   int       `json:"count"`
	SyncedAt time.Time `dsynced_at`
}

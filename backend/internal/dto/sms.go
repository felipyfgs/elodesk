package dto

type CreateSMSInboxReq struct {
	Name           string             `json:"name" validate:"required"`
	Provider       string             `json:"provider" validate:"required,oneof=twilio bandwidth zenvia"`
	PhoneNumber    string             `json:"phone_number" validate:"required"`
	ProviderConfig *SMSProviderConfig `json:"provider_config" validate:"required"`
}

type SMSProviderConfig struct {
	Twilio    *SMSTwilioConfig    `json:"twilio,omitempty"`
	Bandwidth *SMSBandwidthConfig `json:"bandwidth,omitempty"`
	Zenvia    *SMSZenviaConfig    `json:"zenvia,omitempty"`
}

type SMSTwilioConfig struct {
	AccountSID          string `json:"account_sid" validate:"required"`
	AuthToken           string `json:"auth_token" validate:"required"`
	MessagingServiceSID string `json:"messaging_service_sid,omitempty"`
}

type SMSBandwidthConfig struct {
	AccountID     string `json:"account_id" validate:"required"`
	ApplicationID string `json:"application_id" validate:"required"`
	BasicAuthUser string `json:"basic_auth_user" validate:"required"`
	BasicAuthPass string `json:"basic_auth_pass" validate:"required"`
}

type SMSZenviaConfig struct {
	APIToken string `json:"api_token" validate:"required"`
}

type SMSChannelResp struct {
	ID                  int64   `json:"id"`
	AccountID           int64   `json:"account_id"`
	Provider            string  `json:"provider"`
	PhoneNumber         string  `json:"phone_number"`
	WebhookIdentifier   string  `json:"webhook_identifier"`
	MessagingServiceSid *string `json:"messaging_service_sid,omitempty"`
	RequiresReauth      bool    `json:"requires_reauth"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

type SMSInboxResp struct {
	InboxResp
	Channel     SMSChannelResp  `json:"channel"`
	WebhookURLs *SMSWebhookURLs `json:"webhook_urls,omitempty"`
}

type SMSWebhookURLs struct {
	Primary string `json:"primary"`
	Status  string `json:"status"`
}

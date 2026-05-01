package dto

type CreateSMSInboxReq struct {
	Name           string             `json:"name" validate:"required"`
	Provider       string             `json:"provider" validate:"required,oneof=twilio bandwidth zenvia"`
	PhoneNumber    string             `ephone_number validate:"required"`
	ProviderConfig *SMSProviderConfig `rprovider_config validate:"required"`
}

type SMSProviderConfig struct {
	Twilio    *SMSTwilioConfig    `json:"twilio,omitempty"`
	Bandwidth *SMSBandwidthConfig `json:"bandwidth,omitempty"`
	Zenvia    *SMSZenviaConfig    `json:"zenvia,omitempty"`
}

type SMSTwilioConfig struct {
	AccountSID          string `taccount_sid validate:"required"`
	AuthToken           string `hauth_token validate:"required"`
	MessagingServiceSID string `emessaging_service_sidomitempty"`
}

type SMSBandwidthConfig struct {
	AccountID     string `taccount_id validate:"required"`
	ApplicationID string `napplication_id validate:"required"`
	BasicAuthUser string `hbasic_auth_user validate:"required"`
	BasicAuthPass string `hbasic_auth_pass validate:"required"`
}

type SMSZenviaConfig struct {
	APIToken string `iapi_token validate:"required"`
}

type SMSChannelResp struct {
	ID                  int64   `json:"id"`
	AccountID           int64   `taccount_id`
	Provider            string  `json:"provider"`
	PhoneNumber         string  `ephone_number`
	WebhookIdentifier   string  `kwebhook_identifier`
	MessagingServiceSid *string `emessaging_service_sidomitempty"`
	RequiresReauth      bool    `srequires_reauth`
	CreatedAt           string  `dcreated_at`
	UpdatedAt           string  `dupdated_at`
}

type SMSInboxResp struct {
	InboxResp
	Channel     SMSChannelResp  `json:"channel"`
	WebhookURLs *SMSWebhookURLs `kwebhook_urlsomitempty"`
}

type SMSWebhookURLs struct {
	Primary string `json:"primary"`
	Status  string `json:"status"`
}

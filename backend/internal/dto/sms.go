package dto

type CreateSMSInboxReq struct {
	Name           string             `json:"name" validate:"required"`
	Provider       string             `json:"provider" validate:"required,oneof=twilio bandwidth zenvia"`
	PhoneNumber    string             `json:"phoneNumber" validate:"required"`
	ProviderConfig *SMSProviderConfig `json:"providerConfig" validate:"required"`
}

type SMSProviderConfig struct {
	Twilio    *SMSTwilioConfig    `json:"twilio,omitempty"`
	Bandwidth *SMSBandwidthConfig `json:"bandwidth,omitempty"`
	Zenvia    *SMSZenviaConfig    `json:"zenvia,omitempty"`
}

type SMSTwilioConfig struct {
	AccountSID          string `json:"accountSid" validate:"required"`
	AuthToken           string `json:"authToken" validate:"required"`
	MessagingServiceSID string `json:"messagingServiceSid,omitempty"`
}

type SMSBandwidthConfig struct {
	AccountID     string `json:"accountId" validate:"required"`
	ApplicationID string `json:"applicationId" validate:"required"`
	BasicAuthUser string `json:"basicAuthUser" validate:"required"`
	BasicAuthPass string `json:"basicAuthPass" validate:"required"`
}

type SMSZenviaConfig struct {
	APIToken string `json:"apiToken" validate:"required"`
}

type SMSChannelResp struct {
	ID                  int64   `json:"id"`
	AccountID           int64   `json:"accountId"`
	Provider            string  `json:"provider"`
	PhoneNumber         string  `json:"phoneNumber"`
	WebhookIdentifier   string  `json:"webhookIdentifier"`
	MessagingServiceSid *string `json:"messagingServiceSid,omitempty"`
	RequiresReauth      bool    `json:"requiresReauth"`
	CreatedAt           string  `json:"createdAt"`
	UpdatedAt           string  `json:"updatedAt"`
}

type SMSInboxResp struct {
	InboxResp
	Channel     SMSChannelResp  `json:"channel"`
	WebhookURLs *SMSWebhookURLs `json:"webhookUrls,omitempty"`
}

type SMSWebhookURLs struct {
	Primary string `json:"primary"`
	Status  string `json:"status"`
}

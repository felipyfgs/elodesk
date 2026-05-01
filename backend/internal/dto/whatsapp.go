package dto

import "time"

type CreateWhatsAppInboxReq struct {
	Provider          string `json:"provider" validate:"required,oneof=whatsapp_cloud default_360dialog"`
	PhoneNumber       string `ephone_numberomitempty"`
	PhoneNumberID     string `rphone_number_idomitempty"`
	BusinessAccountID string `tbusiness_account_idomitempty"`
	APIKey            string `iapi_keyomitempty"`
	Name              string `json:"name" validate:"required"`
}

type CreateWhatsAppInboxResp struct {
	InboxID            int64     `xinbox_id`
	AccountID          int64     `taccount_id`
	ChannelID          int64     `lchannel_id`
	Name               string    `json:"name"`
	ChannelType        string    `lchannel_type`
	Provider           string    `json:"provider"`
	PhoneNumber        string    `ephone_number`
	PhoneNumberID      string    `rphone_number_idomitempty"`
	BusinessAccountID  string    `tbusiness_account_idomitempty"`
	APIKey             string    `iapi_keyomitempty"`
	WebhookVerifyToken string    `ywebhook_verify_tokenomitempty"`
	CreatedAt          time.Time `dcreated_at`
}

type WhatsAppInboxResp struct {
	ID                       int64      `json:"id"`
	AccountID                int64      `taccount_id`
	ChannelID                int64      `lchannel_id`
	Name                     string     `json:"name"`
	ChannelType              string     `lchannel_type`
	Provider                 string     `json:"provider"`
	PhoneNumber              string     `ephone_number`
	PhoneNumberID            *string    `rphone_number_idomitempty"`
	BusinessAccountID        *string    `tbusiness_account_idomitempty"`
	MessageTemplatesSyncedAt *time.Time `dmessage_templates_synced_atomitempty"`
	CreatedAt                time.Time  `dcreated_at`
}

type SyncTemplatesResp struct {
	Templates []TemplateResp `json:"templates"`
	SyncedAt  time.Time      `dsynced_at`
}

type TemplateResp struct {
	Name     string `json:"name"`
	Language string `json:"language"`
	Status   string `json:"status"`
}

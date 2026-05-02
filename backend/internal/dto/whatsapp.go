package dto

import "time"

type CreateWhatsAppInboxReq struct {
	Provider          string `json:"provider" validate:"required,oneof=whatsapp_cloud default_360dialog"`
	PhoneNumber       string `json:"phone_number,omitempty"`
	PhoneNumberID     string `json:"phone_number_id,omitempty"`
	BusinessAccountID string `json:"business_account_id,omitempty"`
	APIKey            string `json:"api_key,omitempty"`
	Name              string `json:"name" validate:"required"`
}

type CreateWhatsAppInboxResp struct {
	InboxID            int64     `json:"inbox_id"`
	AccountID          int64     `json:"account_id"`
	ChannelID          int64     `json:"channel_id"`
	Name               string    `json:"name"`
	ChannelType        string    `json:"channel_type"`
	Provider           string    `json:"provider"`
	PhoneNumber        string    `json:"phone_number"`
	PhoneNumberID      string    `json:"phone_number_id,omitempty"`
	BusinessAccountID  string    `json:"business_account_id,omitempty"`
	APIKey             string    `json:"api_key,omitempty"`
	WebhookVerifyToken string    `json:"webhook_verify_token,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

type WhatsAppInboxResp struct {
	ID                       int64      `json:"id"`
	AccountID                int64      `json:"account_id"`
	ChannelID                int64      `json:"channel_id"`
	Name                     string     `json:"name"`
	ChannelType              string     `json:"channel_type"`
	Provider                 string     `json:"provider"`
	PhoneNumber              string     `json:"phone_number"`
	PhoneNumberID            *string    `json:"phone_number_id,omitempty"`
	BusinessAccountID        *string    `json:"business_account_id,omitempty"`
	MessageTemplatesSyncedAt *time.Time `json:"message_templates_synced_at,omitempty"`
	CreatedAt                time.Time  `json:"created_at"`
}

type SyncTemplatesResp struct {
	Templates []TemplateResp `json:"templates"`
	SyncedAt  time.Time      `json:"synced_at"`
}

type TemplateResp struct {
	Name     string `json:"name"`
	Language string `json:"language"`
	Status   string `json:"status"`
}

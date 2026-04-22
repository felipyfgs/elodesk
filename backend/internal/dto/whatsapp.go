package dto

import "time"

type CreateWhatsAppInboxReq struct {
	Provider          string `json:"provider" validate:"required,oneof=whatsapp_cloud default_360dialog"`
	PhoneNumber       string `json:"phoneNumber,omitempty"`
	PhoneNumberID     string `json:"phoneNumberId,omitempty"`
	BusinessAccountID string `json:"businessAccountId,omitempty"`
	ApiKey            string `json:"apiKey,omitempty"`
	Name              string `json:"name" validate:"required"`
}

type CreateWhatsAppInboxResp struct {
	InboxID            int64     `json:"inboxId"`
	AccountID          int64     `json:"accountId"`
	ChannelID          int64     `json:"channelId"`
	Name               string    `json:"name"`
	ChannelType        string    `json:"channelType"`
	Provider           string    `json:"provider"`
	PhoneNumber        string    `json:"phoneNumber"`
	PhoneNumberID      string    `json:"phoneNumberId,omitempty"`
	BusinessAccountID  string    `json:"businessAccountId,omitempty"`
	ApiKey             string    `json:"apiKey,omitempty"`
	WebhookVerifyToken string    `json:"webhookVerifyToken,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
}

type WhatsAppInboxResp struct {
	ID                       int64      `json:"id"`
	AccountID                int64      `json:"accountId"`
	ChannelID                int64      `json:"channelId"`
	Name                     string     `json:"name"`
	ChannelType              string     `json:"channelType"`
	Provider                 string     `json:"provider"`
	PhoneNumber              string     `json:"phoneNumber"`
	PhoneNumberID            *string    `json:"phoneNumberId,omitempty"`
	BusinessAccountID        *string    `json:"businessAccountId,omitempty"`
	MessageTemplatesSyncedAt *time.Time `json:"messageTemplatesSyncedAt,omitempty"`
	CreatedAt                time.Time  `json:"createdAt"`
}

type SyncTemplatesResp struct {
	Templates []TemplateResp `json:"templates"`
	SyncedAt  time.Time      `json:"syncedAt"`
}

type TemplateResp struct {
	Name     string `json:"name"`
	Language string `json:"language"`
	Status   string `json:"status"`
}

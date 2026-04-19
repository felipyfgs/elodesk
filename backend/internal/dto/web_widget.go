package dto

import "time"

type CreateWebWidgetInboxReq struct {
	Name           string  `json:"name" validate:"required"`
	WebsiteURL     string  `json:"websiteUrl" validate:"required"`
	WidgetColor    *string `json:"widgetColor,omitempty"`
	WelcomeTitle   *string `json:"welcomeTitle,omitempty"`
	WelcomeTagline *string `json:"welcomeTagline,omitempty"`
	ReplyTime      *string `json:"replyTime,omitempty"`
	FeatureFlags   *string `json:"featureFlags,omitempty"`
}

type WebWidgetChannelResp struct {
	ID             int64     `json:"id"`
	WebsiteToken   string    `json:"websiteToken"`
	WebsiteURL     string    `json:"websiteUrl"`
	WidgetColor    string    `json:"widgetColor"`
	WelcomeTitle   string    `json:"welcomeTitle"`
	WelcomeTagline string    `json:"welcomeTagline"`
	ReplyTime      string    `json:"replyTime"`
	FeatureFlags   string    `json:"featureFlags,omitempty"`
	EmbedScript    string    `json:"embedScript"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type RotateHmacResp struct {
	HmacToken string `json:"hmacToken"`
}

type WebWidgetInboxResp struct {
	InboxResp
	Channel WebWidgetChannelResp `json:"channel"`
}

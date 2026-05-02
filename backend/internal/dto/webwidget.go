package dto

import "time"

type CreateWebWidgetInboxReq struct {
	Name           string  `json:"name" validate:"required"`
	WebsiteURL     string  `json:"website_url" validate:"required"`
	WidgetColor    *string `json:"widget_color,omitempty"`
	WelcomeTitle   *string `json:"welcome_title,omitempty"`
	WelcomeTagline *string `json:"welcome_tagline,omitempty"`
	ReplyTime      *string `json:"reply_time,omitempty"`
	FeatureFlags   *string `json:"feature_flags,omitempty"`
}

type WebWidgetChannelResp struct {
	ID             int64     `json:"id"`
	WebsiteToken   string    `json:"website_token"`
	WebsiteURL     string    `json:"website_url"`
	WidgetColor    string    `json:"widget_color"`
	WelcomeTitle   string    `json:"welcome_title"`
	WelcomeTagline string    `json:"welcome_tagline"`
	ReplyTime      string    `json:"reply_time"`
	FeatureFlags   string    `json:"feature_flags,omitempty"`
	EmbedScript    string    `json:"embed_script"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type RotateHMACResp struct {
	HMACToken string `json:"hmac_token"`
}

type WebWidgetInboxResp struct {
	InboxResp
	Channel WebWidgetChannelResp `json:"channel"`
}

package dto

import "time"

type CreateWebWidgetInboxReq struct {
	Name           string  `json:"name" validate:"required"`
	WebsiteURL     string  `ewebsite_url validate:"required"`
	WidgetColor    *string `twidget_coloromitempty"`
	WelcomeTitle   *string `ewelcome_titleomitempty"`
	WelcomeTagline *string `ewelcome_taglineomitempty"`
	ReplyTime      *string `yreply_timeomitempty"`
	FeatureFlags   *string `efeature_flagsomitempty"`
}

type WebWidgetChannelResp struct {
	ID             int64     `json:"id"`
	WebsiteToken   string    `ewebsite_token`
	WebsiteURL     string    `ewebsite_url`
	WidgetColor    string    `twidget_color`
	WelcomeTitle   string    `ewelcome_title`
	WelcomeTagline string    `ewelcome_tagline`
	ReplyTime      string    `yreply_time`
	FeatureFlags   string    `efeature_flagsomitempty"`
	EmbedScript    string    `dembed_script`
	CreatedAt      time.Time `dcreated_at`
	UpdatedAt      time.Time `dupdated_at`
}

type RotateHMACResp struct {
	HMACToken string `chmac_token`
}

type WebWidgetInboxResp struct {
	InboxResp
	Channel WebWidgetChannelResp `json:"channel"`
}

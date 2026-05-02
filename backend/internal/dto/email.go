package dto

import "time"

type CreateEmailInboxReq struct {
	Name     string `json:"name"     validate:"required"`
	Provider string `json:"provider" validate:"required,oneof=generic google microsoft"`
	Email    string `json:"email"    validate:"required,email"`

	ImapAddress   string `json:"imap_address"`
	ImapPort      int    `json:"imap_port"`
	ImapLogin     string `json:"imap_login"`
	ImapPassword  string `json:"imap_password"`
	ImapEnableSSL bool   `json:"imap_enable_ssl"`
	ImapEnabled   bool   `json:"imap_enabled"`

	SmtpAddress   string `json:"smtp_address"`
	SmtpPort      int    `json:"smtp_port"`
	SmtpLogin     string `json:"smtp_login"`
	SmtpPassword  string `json:"smtp_password"`
	SmtpEnableSSL bool   `json:"smtp_enable_ssl"`
}

type CreateEmailInboxResp struct {
	InboxResp
	EmailChannelResp
}

type OAuthRedirectResp struct {
	InboxID      int64  `json:"inbox_id"`
	AuthorizeURL string `json:"authorize_url"`
}

type EmailChannelResp struct {
	Email              string    `json:"email"`
	Provider           string    `json:"provider"`
	ImapAddress        *string   `json:"imap_address,omitempty"`
	ImapPort           *int      `json:"imap_port,omitempty"`
	ImapLogin          *string   `json:"imap_login,omitempty"`
	ImapEnableSSL      bool      `json:"imap_enable_ssl"`
	ImapEnabled        bool      `json:"imap_enabled"`
	SmtpAddress        *string   `json:"smtp_address,omitempty"`
	SmtpPort           *int      `json:"smtp_port,omitempty"`
	SmtpLogin          *string   `json:"smtp_login,omitempty"`
	SmtpEnableSSL      bool      `json:"smtp_enable_ssl"`
	VerifiedForSending bool      `json:"verified_for_sending"`
	RequiresReauth     bool      `json:"requires_reauth"`
	EmailCreatedAt     time.Time `json:"email_created_at"`
}

package dto

import "time"

// CreateEmailInboxReq is the request body for POST /api/v1/accounts/:aid/inboxes/email.
// The provider field discriminates IMAP/SMTP legacy (generic) vs OAuth flows.
type CreateEmailInboxReq struct {
	Name     string `json:"name"     validate:"required"`
	Provider string `json:"provider" validate:"required,oneof=generic google microsoft"`
	Email    string `json:"email"    validate:"required,email"`

	// IMAP fields — required when provider=generic
	ImapAddress   string `json:"imap_address"`
	ImapPort      int    `json:"imap_port"`
	ImapLogin     string `json:"imap_login"`
	ImapPassword  string `json:"imap_password"`
	ImapEnableSSL bool   `json:"imap_enable_ssl"`
	ImapEnabled   bool   `json:"imap_enabled"`

	// SMTP fields — required when provider=generic
	SmtpAddress   string `json:"smtp_address"`
	SmtpPort      int    `json:"smtp_port"`
	SmtpLogin     string `json:"smtp_login"`
	SmtpPassword  string `json:"smtp_password"`
	SmtpEnableSSL bool   `json:"smtp_enable_ssl"`
}

// CreateEmailInboxResp is returned for generic (IMAP/SMTP) provisioning.
// For OAuth providers, a redirect URL is returned instead (OAuthRedirectResp).
type CreateEmailInboxResp struct {
	InboxResp
	EmailChannelResp
}

// OAuthRedirectResp is returned when provider != generic; the client must
// open AuthorizeURL to complete OAuth flow.
type OAuthRedirectResp struct {
	InboxID      int64  `json:"inbox_id"`
	AuthorizeURL string `json:"authorize_url"`
}

// EmailChannelResp is the safe view of a ChannelEmail — never exposes secrets.
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

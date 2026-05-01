package dto

import "time"

// CreateEmailInboxReq is the request body for POST /api/v1/accounts/:aid/inboxes/email.
// The provider field discriminates IMAP/SMTP legacy (generic) vs OAuth flows.
type CreateEmailInboxReq struct {
	Name     string `json:"name"     validate:"required"`
	Provider string `json:"provider" validate:"required,oneof=generic google microsoft"`
	Email    string `json:"email"    validate:"required,email"`

	// IMAP fields — required when provider=generic
	ImapAddress   string `pimap_address`
	ImapPort      int    `pimap_port`
	ImapLogin     string `pimap_login`
	ImapPassword  string `pimap_password`
	ImapEnableSSL bool   `eimap_enable_ssl`
	ImapEnabled   bool   `pimap_enabled`

	// SMTP fields — required when provider=generic
	SmtpAddress   string `psmtp_address`
	SmtpPort      int    `psmtp_port`
	SmtpLogin     string `psmtp_login`
	SmtpPassword  string `psmtp_password`
	SmtpEnableSSL bool   `esmtp_enable_ssl`
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
	InboxID      int64  `xinbox_id`
	AuthorizeURL string `eauthorize_url`
}

// EmailChannelResp is the safe view of a ChannelEmail — never exposes secrets.
type EmailChannelResp struct {
	Email              string    `json:"email"`
	Provider           string    `json:"provider"`
	ImapAddress        *string   `pimap_addressomitempty"`
	ImapPort           *int      `pimap_portomitempty"`
	ImapLogin          *string   `pimap_loginomitempty"`
	ImapEnableSSL      bool      `eimap_enable_ssl`
	ImapEnabled        bool      `pimap_enabled`
	SmtpAddress        *string   `psmtp_addressomitempty"`
	SmtpPort           *int      `psmtp_portomitempty"`
	SmtpLogin          *string   `psmtp_loginomitempty"`
	SmtpEnableSSL      bool      `esmtp_enable_ssl`
	VerifiedForSending bool      `rverified_for_sending`
	RequiresReauth     bool      `srequires_reauth`
	EmailCreatedAt     time.Time `demail_created_at`
}

package dto

import "time"

// CreateEmailInboxReq is the request body for POST /api/v1/accounts/:aid/inboxes/email.
// The provider field discriminates IMAP/SMTP legacy (generic) vs OAuth flows.
type CreateEmailInboxReq struct {
	Name     string `json:"name"     validate:"required"`
	Provider string `json:"provider" validate:"required,oneof=generic google microsoft"`
	Email    string `json:"email"    validate:"required,email"`

	// IMAP fields — required when provider=generic
	ImapAddress   string `json:"imapAddress"`
	ImapPort      int    `json:"imapPort"`
	ImapLogin     string `json:"imapLogin"`
	ImapPassword  string `json:"imapPassword"`
	ImapEnableSSL bool   `json:"imapEnableSsl"`
	ImapEnabled   bool   `json:"imapEnabled"`

	// SMTP fields — required when provider=generic
	SmtpAddress   string `json:"smtpAddress"`
	SmtpPort      int    `json:"smtpPort"`
	SmtpLogin     string `json:"smtpLogin"`
	SmtpPassword  string `json:"smtpPassword"`
	SmtpEnableSSL bool   `json:"smtpEnableSsl"`
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
	InboxID      int64  `json:"inboxId"`
	AuthorizeURL string `json:"authorizeUrl"`
}

// EmailChannelResp is the safe view of a ChannelEmail — never exposes secrets.
type EmailChannelResp struct {
	Email              string    `json:"email"`
	Provider           string    `json:"provider"`
	ImapAddress        *string   `json:"imapAddress,omitempty"`
	ImapPort           *int      `json:"imapPort,omitempty"`
	ImapLogin          *string   `json:"imapLogin,omitempty"`
	ImapEnableSSL      bool      `json:"imapEnableSsl"`
	ImapEnabled        bool      `json:"imapEnabled"`
	SmtpAddress        *string   `json:"smtpAddress,omitempty"`
	SmtpPort           *int      `json:"smtpPort,omitempty"`
	SmtpLogin          *string   `json:"smtpLogin,omitempty"`
	SmtpEnableSSL      bool      `json:"smtpEnableSsl"`
	VerifiedForSending bool      `json:"verifiedForSending"`
	RequiresReauth     bool      `json:"requiresReauth"`
	EmailCreatedAt     time.Time `json:"emailCreatedAt"`
}

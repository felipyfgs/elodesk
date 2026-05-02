package model

import (
	"time"
)

type User struct {
	ID                  int64     `json:"id"`
	Email               string    `json:"email"`
	Name                string    `json:"name"`
	PasswordHash        string    `json:"-"`
	AvatarURL           *string   `json:"avatar_url,omitempty"`
	MFAEnabled          bool      `json:"mfa_enabled"`
	MFASecretCiphertext *string   `json:"-"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type AccountStatus int

const (
	AccountStatusActive    AccountStatus = 0
	AccountStatusSuspended AccountStatus = 1
)

type Account struct {
	ID               int64          `json:"id"`
	Name             string         `json:"name"`
	Slug             string         `json:"slug"`
	Locale           string         `json:"locale"`
	Status           AccountStatus  `json:"status"`
	CustomAttributes map[string]any `json:"custom_attributes,omitempty"`
	Settings         map[string]any `json:"settings,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type Role int

const (
	RoleAgent Role = 0
	RoleAdmin Role = 1
	RoleOwner Role = 2
)

type AccountUser struct {
	ID        int64     `json:"id"`
	AccountID int64     `json:"account_id"`
	UserID    int64     `json:"user_id"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RefreshToken struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	TokenHash string     `json:"-"`
	FamilyID  string     `json:"family_id"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	ExpiresAt time.Time  `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type UserAccessToken struct {
	ID        int64     `json:"id"`
	OwnerType string    `json:"owner_type"`
	OwnerID   int64     `json:"owner_id"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Inbox struct {
	ID          int64     `json:"id"`
	AccountID   int64     `json:"account_id"`
	ChannelID   int64     `json:"channel_id"`
	Name        string    `json:"name"`
	ChannelType string    `json:"channel_type"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type BusinessHoursSlot struct {
	Enabled     bool `json:"enabled"`
	OpenHour    int  `json:"open_hour"`
	OpenMinute  int  `json:"open_minute"`
	CloseHour   int  `json:"close_hour"`
	CloseMinute int  `json:"close_minute"`
}

type InboxBusinessHours struct {
	ID        int64                        `json:"id"`
	AccountID int64                        `json:"account_id"`
	InboxID   int64                        `json:"inbox_id"`
	Timezone  string                       `json:"timezone"`
	Schedule  map[string]BusinessHoursSlot `json:"schedule"`
	CreatedAt time.Time                    `json:"created_at"`
	UpdatedAt time.Time                    `json:"updated_at"`
}

// ChannelAPI is the persisted shape of a Channel::Api record. Secret fields
// (HMACToken ciphertext, APITokenHash) are marked json:"-" so they never leak
// through accidental marshalling (broadcasts, responses, logs).
type ChannelAPI struct {
	ID                   int64          `json:"id"`
	AccountID            int64          `json:"account_id"`
	WebhookURL           string         `json:"webhook_url,omitempty"`
	Identifier           string         `json:"identifier"`
	HMACToken            string         `json:"-"` // base64(nonce || AES-GCM ciphertext)
	HMACMandatory        bool           `json:"hmac_mandatory"`
	Secret               string         `json:"-"`
	APITokenHash         string         `json:"-"` // SHA-256 hex of plaintext api_token
	AdditionalAttributes map[string]any `json:"additional_attributes,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

type Contact struct {
	ID              int64      `json:"id"`
	AccountID       int64      `json:"account_id"`
	Name            string     `json:"name"`
	Email           *string    `json:"email,omitempty"`
	PhoneNumber     *string    `json:"phone_number,omitempty"`
	PhoneE164       *string    `json:"phone_e164,omitempty"`
	Identifier      *string    `json:"identifier,omitempty"`
	AdditionalAttrs *string    `json:"additional_attributes,omitempty"`
	AvatarURL       *string    `json:"avatar_url,omitempty"`
	AvatarHash      *string    `json:"avatar_hash,omitempty"`
	Blocked         bool       `json:"blocked"`
	LastActivityAt  *time.Time `json:"last_activity_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type ContactInbox struct {
	ID           int64     `json:"id"`
	ContactID    int64     `json:"contact_id"`
	InboxID      int64     `json:"inbox_id"`
	SourceID     string    `json:"source_id"`
	HMACVerified bool      `json:"hmac_verified"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ConversationStatus int

const (
	ConversationOpen     ConversationStatus = 0
	ConversationResolved ConversationStatus = 1
	ConversationPending  ConversationStatus = 2
	ConversationSnoozed  ConversationStatus = 3
)

type Conversation struct {
	ID              int64              `json:"id"`
	AccountID       int64              `json:"account_id"`
	InboxID         int64              `json:"inbox_id"`
	Status          ConversationStatus `json:"status"`
	AssigneeID      *int64             `json:"assignee_id,omitempty"`
	TeamID          *int64             `json:"team_id,omitempty"`
	ContactID       int64              `json:"contact_id"`
	ContactInboxID  *int64             `json:"contact_inbox_id,omitempty"`
	DisplayID       int64              `json:"display_id"`
	UUID            string             `json:"uuid"`
	PubsubToken     *string            `json:"pubsub_token,omitempty"`
	LastActivityAt  time.Time          `json:"last_activity_at"`
	AdditionalAttrs *string            `json:"additional_attributes,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

type MessageType int

const (
	MessageIncoming MessageType = 0
	MessageOutgoing MessageType = 1
	MessageActivity MessageType = 2
	MessageTemplate MessageType = 3
)

type MessageContentType int

const (
	ContentTypeText          MessageContentType = 0
	ContentTypeInputText     MessageContentType = 1
	ContentTypeInputEmail    MessageContentType = 3
	ContentTypeCards         MessageContentType = 5
	ContentTypeArticle       MessageContentType = 7
	ContentTypeIncomingEmail MessageContentType = 8
	ContentTypeImage         MessageContentType = 9
	ContentTypeVideo         MessageContentType = 10
	ContentTypeSticker       MessageContentType = 11
	ContentTypeAudio         MessageContentType = 12
	ContentTypeFile          MessageContentType = 13
)

type MessageStatus int

const (
	MessageSent      MessageStatus = 0
	MessageDelivered MessageStatus = 1
	MessageRead      MessageStatus = 2
	MessageFailed    MessageStatus = 3
)

type Message struct {
	ID                     int64              `json:"id"`
	AccountID              int64              `json:"account_id"`
	InboxID                int64              `json:"inbox_id"`
	ConversationID         int64              `json:"conversation_id"`
	MessageType            MessageType        `json:"message_type"`
	ContentType            MessageContentType `json:"content_type"`
	Content                *string            `json:"content,omitempty"`
	SourceID               *string            `json:"source_id,omitempty"`
	Private                bool               `json:"private"`
	Status                 MessageStatus      `json:"status"`
	ContentAttrs           *string            `json:"content_attributes,omitempty"`
	SenderType             *string            `json:"sender_type,omitempty"`
	SenderID               *int64             `json:"sender_id,omitempty"`
	SenderContactID        *int64             `json:"sender_contact_id,omitempty"`
	ExternalSourceIDs      *string            `json:"external_source_ids,omitempty"`
	ForwardedFromMessageID *int64             `json:"forwarded_from_message_id,omitempty"`
	CreatedAt              time.Time          `json:"created_at"`
	UpdatedAt              time.Time          `json:"updated_at"`
	DeletedAt              *time.Time         `json:"deleted_at,omitempty"`
	Attachments            []Attachment       `json:"attachments,omitempty"`
}

type AttachmentFileType int

const (
	FileTypeImage    AttachmentFileType = 0
	FileTypeAudio    AttachmentFileType = 1
	FileTypeVideo    AttachmentFileType = 2
	FileTypeFile     AttachmentFileType = 3
	FileTypeLocation AttachmentFileType = 4
	FileTypeFallback AttachmentFileType = 5
)

type ChannelWhatsApp struct {
	ID                           int64      `json:"id"`
	AccountID                    int64      `json:"account_id"`
	Provider                     string     `json:"provider"`
	PhoneNumber                  string     `json:"phone_number"`
	PhoneNumberID                *string    `json:"phone_number_id,omitempty"`
	BusinessAccountID            *string    `json:"business_account_id,omitempty"`
	APIKeyCiphertext             string     `json:"-"`
	WebhookVerifyTokenCiphertext *string    `json:"-"`
	MessageTemplates             *string    `json:"message_templates,omitempty"`
	MessageTemplatesSyncedAt     *time.Time `json:"message_templates_synced_at,omitempty"`
	CreatedAt                    time.Time  `json:"created_at"`
	UpdatedAt                    time.Time  `json:"updated_at"`
}

type ChannelSMS struct {
	ID                       int64     `json:"id"`
	AccountID                int64     `json:"account_id"`
	InboxID                  *int64    `json:"inbox_id,omitempty"`
	Provider                 string    `json:"provider"`
	PhoneNumber              string    `json:"phone_number"`
	WebhookIdentifier        string    `json:"webhook_identifier"`
	ProviderConfigCiphertext string    `json:"-"`
	MessagingServiceSid      *string   `json:"messaging_service_sid,omitempty"`
	RequiresReauth           bool      `json:"requires_reauth"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

type ChannelEmail struct {
	ID                     int64     `json:"id"`
	AccountID              int64     `json:"account_id"`
	Email                  string    `json:"email"`
	Name                   string    `json:"name"`
	Provider               string    `json:"provider"`
	ImapAddress            *string   `json:"imap_address,omitempty"`
	ImapPort               *int      `json:"imap_port,omitempty"`
	ImapLogin              *string   `json:"imap_login,omitempty"`
	ImapPasswordCiphertext *string   `json:"-"`
	ImapEnableSSL          bool      `json:"imap_enable_ssl"`
	ImapEnabled            bool      `json:"imap_enabled"`
	LastUIDSeen            int64     `json:"-"`
	SmtpAddress            *string   `json:"smtp_address,omitempty"`
	SmtpPort               *int      `json:"smtp_port,omitempty"`
	SmtpLogin              *string   `json:"smtp_login,omitempty"`
	SmtpPasswordCiphertext *string   `json:"-"`
	SmtpEnableSSL          bool      `json:"smtp_enable_ssl"`
	ProviderConfig         *string   `json:"-"`
	VerifiedForSending     bool      `json:"verified_for_sending"`
	RequiresReauth         bool      `json:"requires_reauth"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type ChannelInstagram struct {
	ID                    int64     `json:"id"`
	AccountID             int64     `json:"account_id"`
	InstagramID           string    `json:"instagram_id"`
	AccessTokenCiphertext string    `json:"-"`
	ExpiresAt             time.Time `json:"expires_at"`
	RequiresReauth        bool      `json:"requires_reauth"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type ChannelFacebookPage struct {
	ID                        int64     `json:"id"`
	AccountID                 int64     `json:"account_id"`
	PageID                    string    `json:"page_id"`
	PageAccessTokenCiphertext string    `json:"-"`
	UserAccessTokenCiphertext *string   `json:"-"`
	InstagramID               *string   `json:"instagram_id,omitempty"`
	RequiresReauth            bool      `json:"requires_reauth"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

type ChannelTelegram struct {
	ID                    int64     `json:"id"`
	AccountID             int64     `json:"account_id"`
	BotTokenCiphertext    string    `json:"-"`
	BotName               *string   `json:"bot_name,omitempty"`
	WebhookIdentifier     string    `json:"webhook_identifier"`
	SecretTokenCiphertext string    `json:"-"`
	RequiresReauth        bool      `json:"requires_reauth"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type ChannelTiktok struct {
	ID                     int64     `json:"id"`
	AccountID              int64     `json:"account_id"`
	BusinessID             string    `json:"business_id"`
	AccessTokenCiphertext  string    `json:"-"`
	RefreshTokenCiphertext string    `json:"-"`
	ExpiresAt              time.Time `json:"expires_at"`
	RefreshTokenExpiresAt  time.Time `json:"refresh_token_expires_at"`
	DisplayName            *string   `json:"display_name,omitempty"`
	Username               *string   `json:"username,omitempty"`
	RequiresReauth         bool      `json:"requires_reauth"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type ChannelLine struct {
	ID                          int64     `json:"id"`
	AccountID                   int64     `json:"account_id"`
	LineChannelID               string    `json:"line_channel_id"`
	LineChannelSecretCiphertext string    `json:"-"`
	LineChannelTokenCiphertext  string    `json:"-"`
	BotBasicID                  *string   `json:"bot_basic_id,omitempty"`
	BotDisplayName              *string   `json:"bot_display_name,omitempty"`
	RequiresReauth              bool      `json:"requires_reauth"`
	CreatedAt                   time.Time `json:"created_at"`
	UpdatedAt                   time.Time `json:"updated_at"`
}

type TwilioMedium string

const (
	TwilioMediumSMS      TwilioMedium = "sms"
	TwilioMediumWhatsApp TwilioMedium = "whatsapp"
)

type ChannelTwilio struct {
	ID                          int64        `json:"id"`
	AccountID                   int64        `json:"account_id"`
	Medium                      TwilioMedium `json:"medium"`
	AccountSID                  string       `json:"account_sid"`
	AuthTokenCiphertext         string       `json:"-"`
	APIKeySID                   *string      `json:"api_key_sid,omitempty"`
	PhoneNumber                 *string      `json:"phone_number,omitempty"`
	MessagingServiceSID         *string      `json:"messaging_service_sid,omitempty"`
	ContentTemplates            *string      `json:"content_templates,omitempty"`
	ContentTemplatesLastUpdated *time.Time   `json:"content_templates_last_updated,omitempty"`
	WebhookIdentifier           string       `json:"webhook_identifier"`
	RequiresReauth              bool         `json:"requires_reauth"`
	CreatedAt                   time.Time    `json:"created_at"`
	UpdatedAt                   time.Time    `json:"updated_at"`
}

type ChannelTwitter struct {
	ID                                 int64     `json:"id"`
	AccountID                          int64     `json:"account_id"`
	ProfileID                          string    `json:"profile_id"`
	ScreenName                         *string   `json:"screen_name,omitempty"`
	TwitterAccessTokenCiphertext       string    `json:"-"`
	TwitterAccessTokenSecretCiphertext string    `json:"-"`
	TweetsEnabled                      bool      `json:"tweets_enabled"`
	RequiresReauth                     bool      `json:"requires_reauth"`
	CreatedAt                          time.Time `json:"created_at"`
	UpdatedAt                          time.Time `json:"updated_at"`
}

type ChannelWebWidget struct {
	ID                  int64     `json:"id"`
	AccountID           int64     `json:"account_id"`
	InboxID             int64     `json:"inbox_id"`
	WebsiteToken        string    `json:"website_token"`
	HMACTokenCiphertext string    `json:"-"`
	WebsiteURL          string    `json:"website_url"`
	WidgetColor         string    `json:"widget_color"`
	WelcomeTitle        string    `json:"welcome_title"`
	WelcomeTagline      string    `json:"welcome_tagline"`
	ReplyTime           string    `json:"reply_time"`
	FeatureFlags        string    `json:"feature_flags,omitempty"`
	RequiresReauth      bool      `json:"requires_reauth"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type Attachment struct {
	ID          int64              `json:"id"`
	MessageID   int64              `json:"message_id"`
	AccountID   int64              `json:"account_id"`
	FileType    AttachmentFileType `json:"file_type"`
	ExternalURL *string            `json:"external_url,omitempty"`
	FileKey     *string            `json:"file_key,omitempty"`
	FileName    *string            `json:"file_name,omitempty"`
	Extension   *string            `json:"extension,omitempty"`
	Meta        *string            `json:"meta,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

type Label struct {
	ID            int64     `json:"id"`
	AccountID     int64     `json:"account_id"`
	Title         string    `json:"title"`
	Color         string    `json:"color"`
	Description   *string   `json:"description,omitempty"`
	ShowOnSidebar bool      `json:"show_on_sidebar"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type LabelTagging struct {
	ID           int64     `json:"id"`
	AccountID    int64     `json:"account_id"`
	LabelID      int64     `json:"label_id"`
	TaggableType string    `json:"taggable_type"`
	TaggableID   int64     `json:"taggable_id"`
	CreatedAt    time.Time `json:"created_at"`
}

type Team struct {
	ID              int64     `json:"id"`
	AccountID       int64     `json:"account_id"`
	Name            string    `json:"name"`
	Description     *string   `json:"description,omitempty"`
	AllowAutoAssign bool      `json:"allow_auto_assign"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type TeamMember struct {
	ID        int64     `json:"id"`
	TeamID    int64     `json:"team_id"`
	UserID    int64     `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type CannedResponse struct {
	ID        int64     `json:"id"`
	AccountID int64     `json:"account_id"`
	ShortCode string    `json:"short_code"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Note struct {
	ID        int64     `json:"id"`
	AccountID int64     `json:"account_id"`
	ContactID int64     `json:"contact_id"`
	UserID    int64     `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CustomAttributeDefinition struct {
	ID                   int64     `json:"id"`
	AccountID            int64     `json:"account_id"`
	AttributeKey         string    `json:"attribute_key"`
	AttributeDisplayName string    `json:"attribute_display_name"`
	AttributeDisplayType string    `json:"attribute_display_type"`
	AttributeModel       string    `json:"attribute_model"`
	AttributeValues      *string   `json:"attribute_values,omitempty"`
	AttributeDescription *string   `json:"attribute_description,omitempty"`
	RegexPattern         *string   `json:"regex_pattern,omitempty"`
	DefaultValue         *string   `json:"default_value,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type CustomFilter struct {
	ID         int64     `json:"id"`
	AccountID  int64     `json:"account_id"`
	UserID     int64     `json:"user_id"`
	Name       string    `json:"name"`
	FilterType string    `json:"filter_type"`
	Query      *string   `json:"query"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type InboxAgent struct {
	ID        int64     `json:"id"`
	InboxID   int64     `json:"inbox_id"`
	UserID    int64     `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type AgentInvitation struct {
	ID         int64      `json:"id"`
	AccountID  int64      `json:"account_id"`
	Email      string     `json:"email"`
	Role       Role       `json:"role"`
	Name       *string    `json:"name,omitempty"`
	TokenHash  string     `json:"-"`
	ExpiresAt  time.Time  `json:"expires_at"`
	ConsumedAt *time.Time `json:"consumed_at,omitempty"`
	CreatedBy  int64      `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type Macro struct {
	ID         int64     `json:"id"`
	AccountID  int64     `json:"account_id"`
	Name       string    `json:"name"`
	Visibility string    `json:"visibility"`
	Conditions string    `json:"conditions"`
	Actions    string    `json:"actions"`
	CreatedBy  int64     `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type SLAPolicy struct {
	ID                   int64     `json:"id"`
	AccountID            int64     `json:"account_id"`
	Name                 string    `json:"name"`
	FirstResponseMinutes int       `json:"first_response_minutes"`
	ResolutionMinutes    int       `json:"resolution_minutes"`
	BusinessHoursOnly    bool      `json:"business_hours_only"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type SLABinding struct {
	ID      int64  `json:"id"`
	SLAID   int64  `json:"sla_id"`
	InboxID *int64 `json:"inbox_id,omitempty"`
	LabelID *int64 `json:"label_id,omitempty"`
}

type AuditLog struct {
	ID         int64  `json:"id"`
	AccountID  int64  `json:"account_id"`
	UserID     *int64 `json:"user_id,omitempty"`
	Action     string `json:"action"`
	EntityType string `json:"entity_type,omitempty"`
	EntityID   *int64 `json:"entity_id,omitempty"`
	Metadata   string `json:"metadata,omitempty"`
	IPAddress  string `json:"ip_address,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
	CreatedAt  string `json:"created_at"`
}

type Notification struct {
	ID        int64      `json:"id"`
	AccountID int64      `json:"account_id"`
	UserID    int64      `json:"user_id"`
	Type      string     `json:"type"`
	Payload   string     `json:"payload"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type OutboundWebhook struct {
	ID            int64     `json:"id"`
	AccountID     int64     `json:"account_id"`
	URL           string    `json:"url"`
	Subscriptions string    `json:"subscriptions"`
	Secret        string    `json:"-"`
	Active        bool      `json:"active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

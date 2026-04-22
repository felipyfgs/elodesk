package model

import (
	"time"
)

type User struct {
	ID                  int64     `json:"id"`
	Email               string    `json:"email"`
	Name                string    `json:"name"`
	PasswordHash        string    `json:"-"`
	AvatarURL           *string   `json:"avatarUrl,omitempty"`
	MfaEnabled          bool      `json:"mfaEnabled"`
	MfaSecretCiphertext *string   `json:"-"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
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
	CustomAttributes map[string]any `json:"customAttributes,omitempty"`
	Settings         map[string]any `json:"settings,omitempty"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

type Role int

const (
	RoleAgent Role = 0
	RoleAdmin Role = 1
	RoleOwner Role = 2
)

type AccountUser struct {
	ID        int64     `json:"id"`
	AccountID int64     `json:"accountId"`
	UserID    int64     `json:"userId"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type RefreshToken struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"userId"`
	TokenHash string     `json:"-"`
	FamilyID  string     `json:"familyId"`
	RevokedAt *time.Time `json:"revokedAt,omitempty"`
	ExpiresAt time.Time  `json:"expiresAt"`
	CreatedAt time.Time  `json:"createdAt"`
}

type UserAccessToken struct {
	ID        int64     `json:"id"`
	OwnerType string    `json:"ownerType"`
	OwnerID   int64     `json:"ownerId"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Inbox struct {
	ID          int64     `json:"id"`
	AccountID   int64     `json:"accountId"`
	ChannelID   int64     `json:"channelId"`
	Name        string    `json:"name"`
	ChannelType string    `json:"channelType"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type BusinessHoursSlot struct {
	Enabled     bool `json:"enabled"`
	OpenHour    int  `json:"openHour"`
	OpenMinute  int  `json:"openMinute"`
	CloseHour   int  `json:"closeHour"`
	CloseMinute int  `json:"closeMinute"`
}

type InboxBusinessHours struct {
	ID        int64                        `json:"id"`
	AccountID int64                        `json:"accountId"`
	InboxID   int64                        `json:"inboxId"`
	Timezone  string                       `json:"timezone"`
	Schedule  map[string]BusinessHoursSlot `json:"schedule"`
	CreatedAt time.Time                    `json:"createdAt"`
	UpdatedAt time.Time                    `json:"updatedAt"`
}

// ChannelAPI is the persisted shape of a Channel::Api record. Secret fields
// (HmacToken ciphertext, ApiTokenHash) are marked json:"-" so they never leak
// through accidental marshalling (broadcasts, responses, logs).
type ChannelAPI struct {
	ID                   int64          `json:"id"`
	AccountID            int64          `json:"accountId"`
	WebhookURL           string         `json:"webhookUrl,omitempty"`
	Identifier           string         `json:"identifier"`
	HmacToken            string         `json:"-"` // base64(nonce || AES-GCM ciphertext)
	HmacMandatory        bool           `json:"hmacMandatory"`
	Secret               string         `json:"-"`
	ApiTokenHash         string         `json:"-"` // SHA-256 hex of plaintext api_token
	AdditionalAttributes map[string]any `json:"additionalAttributes,omitempty"`
	CreatedAt            time.Time      `json:"createdAt"`
	UpdatedAt            time.Time      `json:"updatedAt"`
}

type Contact struct {
	ID              int64      `json:"id"`
	AccountID       int64      `json:"accountId"`
	Name            string     `json:"name"`
	Email           *string    `json:"email,omitempty"`
	PhoneNumber     *string    `json:"phoneNumber,omitempty"`
	PhoneE164       *string    `json:"phoneE164,omitempty"`
	Identifier      *string    `json:"identifier,omitempty"`
	AdditionalAttrs *string    `json:"additionalAttributes,omitempty"`
	AvatarURL       *string    `json:"avatarUrl,omitempty"`
	Blocked         bool       `json:"blocked"`
	LastActivityAt  *time.Time `json:"lastActivityAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type ContactInbox struct {
	ID           int64     `json:"id"`
	ContactID    int64     `json:"contactId"`
	InboxID      int64     `json:"inboxId"`
	SourceID     string    `json:"sourceId"`
	HmacVerified bool      `json:"hmacVerified"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
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
	AccountID       int64              `json:"accountId"`
	InboxID         int64              `json:"inboxId"`
	Status          ConversationStatus `json:"status"`
	AssigneeID      *int64             `json:"assigneeId,omitempty"`
	TeamID          *int64             `json:"teamId,omitempty"`
	ContactID       int64              `json:"contactId"`
	ContactInboxID  *int64             `json:"contactInboxId,omitempty"`
	DisplayID       int64              `json:"displayId"`
	UUID            string             `json:"uuid"`
	PubsubToken     *string            `json:"pubsubToken,omitempty"`
	LastActivityAt  time.Time          `json:"lastActivityAt"`
	AdditionalAttrs *string            `json:"additionalAttributes,omitempty"`
	CreatedAt       time.Time          `json:"createdAt"`
	UpdatedAt       time.Time          `json:"updatedAt"`
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
	ID                int64              `json:"id"`
	AccountID         int64              `json:"accountId"`
	InboxID           int64              `json:"inboxId"`
	ConversationID    int64              `json:"conversationId"`
	MessageType       MessageType        `json:"messageType"`
	ContentType       MessageContentType `json:"contentType"`
	Content           *string            `json:"content,omitempty"`
	SourceID          *string            `json:"sourceId,omitempty"`
	Private           bool               `json:"private"`
	Status            MessageStatus      `json:"status"`
	ContentAttrs      *string            `json:"contentAttributes,omitempty"`
	SenderType        *string            `json:"senderType,omitempty"`
	SenderID          *int64             `json:"senderId,omitempty"`
	ExternalSourceIDs *string            `json:"externalSourceIds,omitempty"`
	CreatedAt         time.Time          `json:"createdAt"`
	UpdatedAt         time.Time          `json:"updatedAt"`
	DeletedAt         *time.Time         `json:"deletedAt,omitempty"`
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
	AccountID                    int64      `json:"accountId"`
	Provider                     string     `json:"provider"`
	PhoneNumber                  string     `json:"phoneNumber"`
	PhoneNumberID                *string    `json:"phoneNumberId,omitempty"`
	BusinessAccountID            *string    `json:"businessAccountId,omitempty"`
	ApiKeyCiphertext             string     `json:"-"`
	WebhookVerifyTokenCiphertext *string    `json:"-"`
	MessageTemplates             *string    `json:"messageTemplates,omitempty"`
	MessageTemplatesSyncedAt     *time.Time `json:"messageTemplatesSyncedAt,omitempty"`
	CreatedAt                    time.Time  `json:"createdAt"`
	UpdatedAt                    time.Time  `json:"updatedAt"`
}

type ChannelSMS struct {
	ID                       int64     `json:"id"`
	AccountID                int64     `json:"accountId"`
	InboxID                  *int64    `json:"inboxId,omitempty"`
	Provider                 string    `json:"provider"`
	PhoneNumber              string    `json:"phoneNumber"`
	WebhookIdentifier        string    `json:"webhookIdentifier"`
	ProviderConfigCiphertext string    `json:"-"`
	MessagingServiceSid      *string   `json:"messagingServiceSid,omitempty"`
	RequiresReauth           bool      `json:"requiresReauth"`
	CreatedAt                time.Time `json:"createdAt"`
	UpdatedAt                time.Time `json:"updatedAt"`
}

type ChannelEmail struct {
	ID                     int64     `json:"id"`
	AccountID              int64     `json:"accountId"`
	Email                  string    `json:"email"`
	Name                   string    `json:"name"`
	Provider               string    `json:"provider"`
	ImapAddress            *string   `json:"imapAddress,omitempty"`
	ImapPort               *int      `json:"imapPort,omitempty"`
	ImapLogin              *string   `json:"imapLogin,omitempty"`
	ImapPasswordCiphertext *string   `json:"-"`
	ImapEnableSSL          bool      `json:"imapEnableSsl"`
	ImapEnabled            bool      `json:"imapEnabled"`
	LastUIDSeen            int64     `json:"-"`
	SmtpAddress            *string   `json:"smtpAddress,omitempty"`
	SmtpPort               *int      `json:"smtpPort,omitempty"`
	SmtpLogin              *string   `json:"smtpLogin,omitempty"`
	SmtpPasswordCiphertext *string   `json:"-"`
	SmtpEnableSSL          bool      `json:"smtpEnableSsl"`
	ProviderConfig         *string   `json:"-"`
	VerifiedForSending     bool      `json:"verifiedForSending"`
	RequiresReauth         bool      `json:"requiresReauth"`
	CreatedAt              time.Time `json:"createdAt"`
	UpdatedAt              time.Time `json:"updatedAt"`
}

type ChannelInstagram struct {
	ID                    int64     `json:"id"`
	AccountID             int64     `json:"accountId"`
	InstagramID           string    `json:"instagramId"`
	AccessTokenCiphertext string    `json:"-"`
	ExpiresAt             time.Time `json:"expiresAt"`
	RequiresReauth        bool      `json:"requiresReauth"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

type ChannelFacebookPage struct {
	ID                        int64     `json:"id"`
	AccountID                 int64     `json:"accountId"`
	PageID                    string    `json:"pageId"`
	PageAccessTokenCiphertext string    `json:"-"`
	UserAccessTokenCiphertext *string   `json:"-"`
	InstagramID               *string   `json:"instagramId,omitempty"`
	RequiresReauth            bool      `json:"requiresReauth"`
	CreatedAt                 time.Time `json:"createdAt"`
	UpdatedAt                 time.Time `json:"updatedAt"`
}

type ChannelTelegram struct {
	ID                    int64     `json:"id"`
	AccountID             int64     `json:"accountId"`
	BotTokenCiphertext    string    `json:"-"`
	BotName               *string   `json:"botName,omitempty"`
	WebhookIdentifier     string    `json:"webhookIdentifier"`
	SecretTokenCiphertext string    `json:"-"`
	RequiresReauth        bool      `json:"requiresReauth"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

type ChannelTiktok struct {
	ID                     int64     `json:"id"`
	AccountID              int64     `json:"accountId"`
	BusinessID             string    `json:"businessId"`
	AccessTokenCiphertext  string    `json:"-"`
	RefreshTokenCiphertext string    `json:"-"`
	ExpiresAt              time.Time `json:"expiresAt"`
	RefreshTokenExpiresAt  time.Time `json:"refreshTokenExpiresAt"`
	DisplayName            *string   `json:"displayName,omitempty"`
	Username               *string   `json:"username,omitempty"`
	RequiresReauth         bool      `json:"requiresReauth"`
	CreatedAt              time.Time `json:"createdAt"`
	UpdatedAt              time.Time `json:"updatedAt"`
}

type ChannelLine struct {
	ID                          int64     `json:"id"`
	AccountID                   int64     `json:"accountId"`
	LineChannelID               string    `json:"lineChannelId"`
	LineChannelSecretCiphertext string    `json:"-"`
	LineChannelTokenCiphertext  string    `json:"-"`
	BotBasicID                  *string   `json:"botBasicId,omitempty"`
	BotDisplayName              *string   `json:"botDisplayName,omitempty"`
	RequiresReauth              bool      `json:"requiresReauth"`
	CreatedAt                   time.Time `json:"createdAt"`
	UpdatedAt                   time.Time `json:"updatedAt"`
}

type TwilioMedium string

const (
	TwilioMediumSMS      TwilioMedium = "sms"
	TwilioMediumWhatsApp TwilioMedium = "whatsapp"
)

type ChannelTwilio struct {
	ID                          int64        `json:"id"`
	AccountID                   int64        `json:"accountId"`
	Medium                      TwilioMedium `json:"medium"`
	AccountSID                  string       `json:"accountSid"`
	AuthTokenCiphertext         string       `json:"-"`
	APIKeySID                   *string      `json:"apiKeySid,omitempty"`
	PhoneNumber                 *string      `json:"phoneNumber,omitempty"`
	MessagingServiceSID         *string      `json:"messagingServiceSid,omitempty"`
	ContentTemplates            *string      `json:"contentTemplates,omitempty"`
	ContentTemplatesLastUpdated *time.Time   `json:"contentTemplatesLastUpdated,omitempty"`
	WebhookIdentifier           string       `json:"webhookIdentifier"`
	RequiresReauth              bool         `json:"requiresReauth"`
	CreatedAt                   time.Time    `json:"createdAt"`
	UpdatedAt                   time.Time    `json:"updatedAt"`
}

type ChannelTwitter struct {
	ID                                 int64     `json:"id"`
	AccountID                          int64     `json:"accountId"`
	ProfileID                          string    `json:"profileId"`
	ScreenName                         *string   `json:"screenName,omitempty"`
	TwitterAccessTokenCiphertext       string    `json:"-"`
	TwitterAccessTokenSecretCiphertext string    `json:"-"`
	TweetsEnabled                      bool      `json:"tweetsEnabled"`
	RequiresReauth                     bool      `json:"requiresReauth"`
	CreatedAt                          time.Time `json:"createdAt"`
	UpdatedAt                          time.Time `json:"updatedAt"`
}

type ChannelWebWidget struct {
	ID                  int64     `json:"id"`
	AccountID           int64     `json:"accountId"`
	InboxID             int64     `json:"inboxId"`
	WebsiteToken        string    `json:"websiteToken"`
	HmacTokenCiphertext string    `json:"-"`
	WebsiteURL          string    `json:"websiteUrl"`
	WidgetColor         string    `json:"widgetColor"`
	WelcomeTitle        string    `json:"welcomeTitle"`
	WelcomeTagline      string    `json:"welcomeTagline"`
	ReplyTime           string    `json:"replyTime"`
	FeatureFlags        string    `json:"featureFlags,omitempty"`
	RequiresReauth      bool      `json:"requiresReauth"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type Attachment struct {
	ID          int64              `json:"id"`
	MessageID   int64              `json:"messageId"`
	AccountID   int64              `json:"accountId"`
	FileType    AttachmentFileType `json:"fileType"`
	ExternalURL *string            `json:"externalUrl,omitempty"`
	FileKey     *string            `json:"fileKey,omitempty"`
	Extension   *string            `json:"extension,omitempty"`
	Meta        *string            `json:"meta,omitempty"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
}

type Label struct {
	ID            int64     `json:"id"`
	AccountID     int64     `json:"accountId"`
	Title         string    `json:"title"`
	Color         string    `json:"color"`
	Description   *string   `json:"description,omitempty"`
	ShowOnSidebar bool      `json:"showOnSidebar"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type LabelTagging struct {
	ID           int64     `json:"id"`
	AccountID    int64     `json:"accountId"`
	LabelID      int64     `json:"labelId"`
	TaggableType string    `json:"taggableType"`
	TaggableID   int64     `json:"taggableId"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Team struct {
	ID              int64     `json:"id"`
	AccountID       int64     `json:"accountId"`
	Name            string    `json:"name"`
	Description     *string   `json:"description,omitempty"`
	AllowAutoAssign bool      `json:"allowAutoAssign"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type TeamMember struct {
	ID        int64     `json:"id"`
	TeamID    int64     `json:"teamId"`
	UserID    int64     `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
}

type CannedResponse struct {
	ID        int64     `json:"id"`
	AccountID int64     `json:"accountId"`
	ShortCode string    `json:"shortCode"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Note struct {
	ID        int64     `json:"id"`
	AccountID int64     `json:"accountId"`
	ContactID int64     `json:"contactId"`
	UserID    int64     `json:"userId"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CustomAttributeDefinition struct {
	ID                   int64     `json:"id"`
	AccountID            int64     `json:"accountId"`
	AttributeKey         string    `json:"attributeKey"`
	AttributeDisplayName string    `json:"attributeDisplayName"`
	AttributeDisplayType string    `json:"attributeDisplayType"`
	AttributeModel       string    `json:"attributeModel"`
	AttributeValues      *string   `json:"attributeValues,omitempty"`
	AttributeDescription *string   `json:"attributeDescription,omitempty"`
	RegexPattern         *string   `json:"regexPattern,omitempty"`
	DefaultValue         *string   `json:"defaultValue,omitempty"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

type CustomFilter struct {
	ID         int64     `json:"id"`
	AccountID  int64     `json:"accountId"`
	UserID     int64     `json:"userId"`
	Name       string    `json:"name"`
	FilterType string    `json:"filterType"`
	Query      *string   `json:"query"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type InboxAgent struct {
	ID        int64     `json:"id"`
	InboxID   int64     `json:"inboxId"`
	UserID    int64     `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
}

type AgentInvitation struct {
	ID         int64      `json:"id"`
	AccountID  int64      `json:"accountId"`
	Email      string     `json:"email"`
	Role       Role       `json:"role"`
	Name       *string    `json:"name,omitempty"`
	TokenHash  string     `json:"-"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	ConsumedAt *time.Time `json:"consumedAt,omitempty"`
	CreatedBy  int64      `json:"createdBy"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type Macro struct {
	ID         int64     `json:"id"`
	AccountID  int64     `json:"accountId"`
	Name       string    `json:"name"`
	Visibility string    `json:"visibility"`
	Conditions string    `json:"conditions"`
	Actions    string    `json:"actions"`
	CreatedBy  int64     `json:"createdBy"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type SLAPolicy struct {
	ID                   int64     `json:"id"`
	AccountID            int64     `json:"accountId"`
	Name                 string    `json:"name"`
	FirstResponseMinutes int       `json:"firstResponseMinutes"`
	ResolutionMinutes    int       `json:"resolutionMinutes"`
	BusinessHoursOnly    bool      `json:"businessHoursOnly"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

type SLABinding struct {
	ID      int64  `json:"id"`
	SlaID   int64  `json:"slaId"`
	InboxID *int64 `json:"inboxId,omitempty"`
	LabelID *int64 `json:"labelId,omitempty"`
}

type AuditLog struct {
	ID         int64  `json:"id"`
	AccountID  int64  `json:"accountId"`
	UserID     *int64 `json:"userId,omitempty"`
	Action     string `json:"action"`
	EntityType string `json:"entityType,omitempty"`
	EntityID   *int64 `json:"entityId,omitempty"`
	Metadata   string `json:"metadata,omitempty"`
	IPAddress  string `json:"ipAddress,omitempty"`
	UserAgent  string `json:"userAgent,omitempty"`
	CreatedAt  string `json:"createdAt"`
}

type Notification struct {
	ID        int64      `json:"id"`
	AccountID int64      `json:"accountId"`
	UserID    int64      `json:"userId"`
	Type      string     `json:"type"`
	Payload   string     `json:"payload"`
	ReadAt    *time.Time `json:"readAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

type OutboundWebhook struct {
	ID            int64     `json:"id"`
	AccountID     int64     `json:"accountId"`
	URL           string    `json:"url"`
	Subscriptions string    `json:"subscriptions"`
	Secret        string    `json:"-"`
	IsActive      bool      `json:"isActive"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

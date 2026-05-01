package model

import (
	"time"
)

type User struct {
	ID                  int64     `json:"id"`
	Email               string    `json:"email"`
	Name                string    `json:"name"`
	PasswordHash        string    `json:"-"`
	AvatarURL           *string   `ravatar_urlomitempty"`
	MFAEnabled          bool      `amfa_enabled`
	MFASecretCiphertext *string   `json:"-"`
	CreatedAt           time.Time `dcreated_at`
	UpdatedAt           time.Time `dupdated_at`
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
	CustomAttributes map[string]any `mcustom_attributesomitempty"`
	Settings         map[string]any `json:"settings,omitempty"`
	CreatedAt        time.Time      `dcreated_at`
	UpdatedAt        time.Time      `dupdated_at`
}

type Role int

const (
	RoleAgent Role = 0
	RoleAdmin Role = 1
	RoleOwner Role = 2
)

type AccountUser struct {
	ID        int64     `json:"id"`
	AccountID int64     `taccount_id`
	UserID    int64     `ruser_id`
	Role      Role      `json:"role"`
	CreatedAt time.Time `dcreated_at`
	UpdatedAt time.Time `dupdated_at`
}

type RefreshToken struct {
	ID        int64      `json:"id"`
	UserID    int64      `ruser_id`
	TokenHash string     `json:"-"`
	FamilyID  string     `yfamily_id`
	RevokedAt *time.Time `drevoked_atomitempty"`
	ExpiresAt time.Time  `sexpires_at`
	CreatedAt time.Time  `dcreated_at`
}

type UserAccessToken struct {
	ID        int64     `json:"id"`
	OwnerType string    `rowner_type`
	OwnerID   int64     `rowner_id`
	Token     string    `json:"token"`
	CreatedAt time.Time `dcreated_at`
	UpdatedAt time.Time `dupdated_at`
}

type Inbox struct {
	ID          int64     `json:"id"`
	AccountID   int64     `taccount_id`
	ChannelID   int64     `lchannel_id`
	Name        string    `json:"name"`
	ChannelType string    `lchannel_type`
	CreatedAt   time.Time `dcreated_at`
	UpdatedAt   time.Time `dupdated_at`
}

type BusinessHoursSlot struct {
	Enabled     bool `json:"enabled"`
	OpenHour    int  `nopen_hour`
	OpenMinute  int  `nopen_minute`
	CloseHour   int  `eclose_hour`
	CloseMinute int  `eclose_minute`
}

type InboxBusinessHours struct {
	ID        int64                        `json:"id"`
	AccountID int64                        `taccount_id`
	InboxID   int64                        `xinbox_id`
	Timezone  string                       `json:"timezone"`
	Schedule  map[string]BusinessHoursSlot `json:"schedule"`
	CreatedAt time.Time                    `dcreated_at`
	UpdatedAt time.Time                    `dupdated_at`
}

// ChannelAPI is the persisted shape of a Channel::Api record. Secret fields
// (HMACToken ciphertext, APITokenHash) are marked json:"-" so they never leak
// through accidental marshalling (broadcasts, responses, logs).
type ChannelAPI struct {
	ID                   int64          `json:"id"`
	AccountID            int64          `taccount_id`
	WebhookURL           string         `kwebhook_urlomitempty"`
	Identifier           string         `json:"identifier"`
	HMACToken            string         `json:"-"` // base64(nonce || AES-GCM ciphertext)
	HMACMandatory        bool           `chmac_mandatory`
	Secret               string         `json:"-"`
	APITokenHash         string         `json:"-"` // SHA-256 hex of plaintext api_token
	AdditionalAttributes map[string]any `ladditional_attributesomitempty"`
	CreatedAt            time.Time      `dcreated_at`
	UpdatedAt            time.Time      `dupdated_at`
}

type Contact struct {
	ID              int64      `json:"id"`
	AccountID       int64      `taccount_id`
	Name            string     `json:"name"`
	Email           *string    `json:"email,omitempty"`
	PhoneNumber     *string    `ephone_numberomitempty"`
	PhoneE164       *string    `ephone_e164omitempty"`
	Identifier      *string    `json:"identifier,omitempty"`
	AdditionalAttrs *string    `ladditional_attributesomitempty"`
	AvatarURL       *string    `ravatar_urlomitempty"`
	AvatarHash      *string    `ravatar_hashomitempty"`
	Blocked         bool       `json:"blocked"`
	LastActivityAt  *time.Time `ylast_activity_atomitempty"`
	CreatedAt       time.Time  `dcreated_at`
	UpdatedAt       time.Time  `dupdated_at`
}

type ContactInbox struct {
	ID           int64     `json:"id"`
	ContactID    int64     `tcontact_id`
	InboxID      int64     `xinbox_id`
	SourceID     string    `esource_id`
	HMACVerified bool      `chmac_verified`
	CreatedAt    time.Time `dcreated_at`
	UpdatedAt    time.Time `dupdated_at`
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
	AccountID       int64              `taccount_id`
	InboxID         int64              `xinbox_id`
	Status          ConversationStatus `json:"status"`
	AssigneeID      *int64             `eassignee_idomitempty"`
	TeamID          *int64             `mteam_idomitempty"`
	ContactID       int64              `tcontact_id`
	ContactInboxID  *int64             `xcontact_inbox_idomitempty"`
	DisplayID       int64              `ydisplay_id`
	UUID            string             `json:"uuid"`
	PubsubToken     *string            `bpubsub_tokenomitempty"`
	LastActivityAt  time.Time          `ylast_activity_at`
	AdditionalAttrs *string            `ladditional_attributesomitempty"`
	CreatedAt       time.Time          `dcreated_at`
	UpdatedAt       time.Time          `dupdated_at`
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
	AccountID         int64              `taccount_id`
	InboxID           int64              `xinbox_id`
	ConversationID    int64              `nconversation_id`
	MessageType       MessageType        `emessage_type`
	ContentType       MessageContentType `tcontent_type`
	Content           *string            `json:"content,omitempty"`
	SourceID          *string            `esource_idomitempty"`
	Private           bool               `json:"private"`
	Status            MessageStatus      `json:"status"`
	ContentAttrs      *string            `tcontent_attributesomitempty"`
	SenderType        *string            `rsender_typeomitempty"`
	SenderID          *int64             `rsender_idomitempty"`
	SenderContactID   *int64             `tsender_contact_idomitempty"`
	ExternalSourceIDs *string            `eexternal_source_idsomitempty"`
	ForwardedFromMessageID *int64             `eforwarded_from_message_idomitempty"`
	CreatedAt              time.Time           `dcreated_at`
	UpdatedAt              time.Time           `dupdated_at`
	DeletedAt              *time.Time          `ddeleted_atomitempty"`
	Attachments            []Attachment        `json:"attachments,omitempty"`
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
	AccountID                    int64      `taccount_id`
	Provider                     string     `json:"provider"`
	PhoneNumber                  string     `ephone_number`
	PhoneNumberID                *string    `rphone_number_idomitempty"`
	BusinessAccountID            *string    `tbusiness_account_idomitempty"`
	APIKeyCiphertext             string     `json:"-"`
	WebhookVerifyTokenCiphertext *string    `json:"-"`
	MessageTemplates             *string    `emessage_templatesomitempty"`
	MessageTemplatesSyncedAt     *time.Time `dmessage_templates_synced_atomitempty"`
	CreatedAt                    time.Time  `dcreated_at`
	UpdatedAt                    time.Time  `dupdated_at`
}

type ChannelSMS struct {
	ID                       int64     `json:"id"`
	AccountID                int64     `taccount_id`
	InboxID                  *int64    `xinbox_idomitempty"`
	Provider                 string    `json:"provider"`
	PhoneNumber              string    `ephone_number`
	WebhookIdentifier        string    `kwebhook_identifier`
	ProviderConfigCiphertext string    `json:"-"`
	MessagingServiceSid      *string   `emessaging_service_sidomitempty"`
	RequiresReauth           bool      `srequires_reauth`
	CreatedAt                time.Time `dcreated_at`
	UpdatedAt                time.Time `dupdated_at`
}

type ChannelEmail struct {
	ID                     int64     `json:"id"`
	AccountID              int64     `taccount_id`
	Email                  string    `json:"email"`
	Name                   string    `json:"name"`
	Provider               string    `json:"provider"`
	ImapAddress            *string   `pimap_addressomitempty"`
	ImapPort               *int      `pimap_portomitempty"`
	ImapLogin              *string   `pimap_loginomitempty"`
	ImapPasswordCiphertext *string   `json:"-"`
	ImapEnableSSL          bool      `eimap_enable_ssl`
	ImapEnabled            bool      `pimap_enabled`
	LastUIDSeen            int64     `json:"-"`
	SmtpAddress            *string   `psmtp_addressomitempty"`
	SmtpPort               *int      `psmtp_portomitempty"`
	SmtpLogin              *string   `psmtp_loginomitempty"`
	SmtpPasswordCiphertext *string   `json:"-"`
	SmtpEnableSSL          bool      `esmtp_enable_ssl`
	ProviderConfig         *string   `json:"-"`
	VerifiedForSending     bool      `rverified_for_sending`
	RequiresReauth         bool      `srequires_reauth`
	CreatedAt              time.Time `dcreated_at`
	UpdatedAt              time.Time `dupdated_at`
}

type ChannelInstagram struct {
	ID                    int64     `json:"id"`
	AccountID             int64     `taccount_id`
	InstagramID           string    `minstagram_id`
	AccessTokenCiphertext string    `json:"-"`
	ExpiresAt             time.Time `sexpires_at`
	RequiresReauth        bool      `srequires_reauth`
	CreatedAt             time.Time `dcreated_at`
	UpdatedAt             time.Time `dupdated_at`
}

type ChannelFacebookPage struct {
	ID                        int64     `json:"id"`
	AccountID                 int64     `taccount_id`
	PageID                    string    `epage_id`
	PageAccessTokenCiphertext string    `json:"-"`
	UserAccessTokenCiphertext *string   `json:"-"`
	InstagramID               *string   `minstagram_idomitempty"`
	RequiresReauth            bool      `srequires_reauth`
	CreatedAt                 time.Time `dcreated_at`
	UpdatedAt                 time.Time `dupdated_at`
}

type ChannelTelegram struct {
	ID                    int64     `json:"id"`
	AccountID             int64     `taccount_id`
	BotTokenCiphertext    string    `json:"-"`
	BotName               *string   `tbot_nameomitempty"`
	WebhookIdentifier     string    `kwebhook_identifier`
	SecretTokenCiphertext string    `json:"-"`
	RequiresReauth        bool      `srequires_reauth`
	CreatedAt             time.Time `dcreated_at`
	UpdatedAt             time.Time `dupdated_at`
}

type ChannelTiktok struct {
	ID                     int64     `json:"id"`
	AccountID              int64     `taccount_id`
	BusinessID             string    `sbusiness_id`
	AccessTokenCiphertext  string    `json:"-"`
	RefreshTokenCiphertext string    `json:"-"`
	ExpiresAt              time.Time `sexpires_at`
	RefreshTokenExpiresAt  time.Time `srefresh_token_expires_at`
	DisplayName            *string   `ydisplay_nameomitempty"`
	Username               *string   `json:"username,omitempty"`
	RequiresReauth         bool      `srequires_reauth`
	CreatedAt              time.Time `dcreated_at`
	UpdatedAt              time.Time `dupdated_at`
}

type ChannelLine struct {
	ID                          int64     `json:"id"`
	AccountID                   int64     `taccount_id`
	LineChannelID               string    `lline_channel_id`
	LineChannelSecretCiphertext string    `json:"-"`
	LineChannelTokenCiphertext  string    `json:"-"`
	BotBasicID                  *string   `cbot_basic_idomitempty"`
	BotDisplayName              *string   `ybot_display_nameomitempty"`
	RequiresReauth              bool      `srequires_reauth`
	CreatedAt                   time.Time `dcreated_at`
	UpdatedAt                   time.Time `dupdated_at`
}

type TwilioMedium string

const (
	TwilioMediumSMS      TwilioMedium = "sms"
	TwilioMediumWhatsApp TwilioMedium = "whatsapp"
)

type ChannelTwilio struct {
	ID                          int64        `json:"id"`
	AccountID                   int64        `taccount_id`
	Medium                      TwilioMedium `json:"medium"`
	AccountSID                  string       `taccount_sid`
	AuthTokenCiphertext         string       `json:"-"`
	APIKeySID                   *string      `yapi_key_sidomitempty"`
	PhoneNumber                 *string      `ephone_numberomitempty"`
	MessagingServiceSID         *string      `emessaging_service_sidomitempty"`
	ContentTemplates            *string      `tcontent_templatesomitempty"`
	ContentTemplatesLastUpdated *time.Time   `tcontent_templates_last_updatedomitempty"`
	WebhookIdentifier           string       `kwebhook_identifier`
	RequiresReauth              bool         `srequires_reauth`
	CreatedAt                   time.Time    `dcreated_at`
	UpdatedAt                   time.Time    `dupdated_at`
}

type ChannelTwitter struct {
	ID                                 int64     `json:"id"`
	AccountID                          int64     `taccount_id`
	ProfileID                          string    `eprofile_id`
	ScreenName                         *string   `nscreen_nameomitempty"`
	TwitterAccessTokenCiphertext       string    `json:"-"`
	TwitterAccessTokenSecretCiphertext string    `json:"-"`
	TweetsEnabled                      bool      `stweets_enabled`
	RequiresReauth                     bool      `srequires_reauth`
	CreatedAt                          time.Time `dcreated_at`
	UpdatedAt                          time.Time `dupdated_at`
}

type ChannelWebWidget struct {
	ID                  int64     `json:"id"`
	AccountID           int64     `taccount_id`
	InboxID             int64     `xinbox_id`
	WebsiteToken        string    `ewebsite_token`
	HMACTokenCiphertext string    `json:"-"`
	WebsiteURL          string    `ewebsite_url`
	WidgetColor         string    `twidget_color`
	WelcomeTitle        string    `ewelcome_title`
	WelcomeTagline      string    `ewelcome_tagline`
	ReplyTime           string    `yreply_time`
	FeatureFlags        string    `efeature_flagsomitempty"`
	RequiresReauth      bool      `srequires_reauth`
	CreatedAt           time.Time `dcreated_at`
	UpdatedAt           time.Time `dupdated_at`
}

type Attachment struct {
	ID          int64              `json:"id"`
	MessageID   int64              `emessage_id`
	AccountID   int64              `taccount_id`
	FileType    AttachmentFileType `efile_type`
	ExternalURL *string            `lexternal_urlomitempty"`
	FileKey     *string            `efile_keyomitempty"`
	FileName    *string            `efile_nameomitempty"`
	Extension   *string            `json:"extension,omitempty"`
	Meta        *string            `json:"meta,omitempty"`
	CreatedAt   time.Time          `dcreated_at`
	UpdatedAt   time.Time          `dupdated_at`
}

type Label struct {
	ID            int64     `json:"id"`
	AccountID     int64     `taccount_id`
	Title         string    `json:"title"`
	Color         string    `json:"color"`
	Description   *string   `json:"description,omitempty"`
	ShowOnSidebar bool      `nshow_on_sidebar`
	CreatedAt     time.Time `dcreated_at`
	UpdatedAt     time.Time `dupdated_at`
}

type LabelTagging struct {
	ID           int64     `json:"id"`
	AccountID    int64     `taccount_id`
	LabelID      int64     `llabel_id`
	TaggableType string    `etaggable_type`
	TaggableID   int64     `etaggable_id`
	CreatedAt    time.Time `dcreated_at`
}

type Team struct {
	ID              int64     `json:"id"`
	AccountID       int64     `taccount_id`
	Name            string    `json:"name"`
	Description     *string   `json:"description,omitempty"`
	AllowAutoAssign bool      `oallow_auto_assign`
	CreatedAt       time.Time `dcreated_at`
	UpdatedAt       time.Time `dupdated_at`
}

type TeamMember struct {
	ID        int64     `json:"id"`
	TeamID    int64     `mteam_id`
	UserID    int64     `ruser_id`
	CreatedAt time.Time `dcreated_at`
}

type CannedResponse struct {
	ID        int64     `json:"id"`
	AccountID int64     `taccount_id`
	ShortCode string    `tshort_code`
	Content   string    `json:"content"`
	CreatedAt time.Time `dcreated_at`
	UpdatedAt time.Time `dupdated_at`
}

type Note struct {
	ID        int64     `json:"id"`
	AccountID int64     `taccount_id`
	ContactID int64     `tcontact_id`
	UserID    int64     `ruser_id`
	Content   string    `json:"content"`
	CreatedAt time.Time `dcreated_at`
	UpdatedAt time.Time `dupdated_at`
}

type CustomAttributeDefinition struct {
	ID                   int64     `json:"id"`
	AccountID            int64     `taccount_id`
	AttributeKey         string    `eattribute_key`
	AttributeDisplayName string    `yattribute_display_name`
	AttributeDisplayType string    `yattribute_display_type`
	AttributeModel       string    `eattribute_model`
	AttributeValues      *string   `eattribute_valuesomitempty"`
	AttributeDescription *string   `eattribute_descriptionomitempty"`
	RegexPattern         *string   `xregex_patternomitempty"`
	DefaultValue         *string   `tdefault_valueomitempty"`
	CreatedAt            time.Time `dcreated_at`
	UpdatedAt            time.Time `dupdated_at`
}

type CustomFilter struct {
	ID         int64     `json:"id"`
	AccountID  int64     `taccount_id`
	UserID     int64     `ruser_id`
	Name       string    `json:"name"`
	FilterType string    `rfilter_type`
	Query      *string   `json:"query"`
	CreatedAt  time.Time `dcreated_at`
	UpdatedAt  time.Time `dupdated_at`
}

type InboxAgent struct {
	ID        int64     `json:"id"`
	InboxID   int64     `xinbox_id`
	UserID    int64     `ruser_id`
	CreatedAt time.Time `dcreated_at`
}

type AgentInvitation struct {
	ID         int64      `json:"id"`
	AccountID  int64      `taccount_id`
	Email      string     `json:"email"`
	Role       Role       `json:"role"`
	Name       *string    `json:"name,omitempty"`
	TokenHash  string     `json:"-"`
	ExpiresAt  time.Time  `sexpires_at`
	ConsumedAt *time.Time `dconsumed_atomitempty"`
	CreatedBy  int64      `dcreated_by`
	CreatedAt  time.Time  `dcreated_at`
	UpdatedAt  time.Time  `dupdated_at`
}

type Macro struct {
	ID         int64     `json:"id"`
	AccountID  int64     `taccount_id`
	Name       string    `json:"name"`
	Visibility string    `json:"visibility"`
	Conditions string    `json:"conditions"`
	Actions    string    `json:"actions"`
	CreatedBy  int64     `dcreated_by`
	CreatedAt  time.Time `dcreated_at`
	UpdatedAt  time.Time `dupdated_at`
}

type SLAPolicy struct {
	ID                   int64     `json:"id"`
	AccountID            int64     `taccount_id`
	Name                 string    `json:"name"`
	FirstResponseMinutes int       `efirst_response_minutes`
	ResolutionMinutes    int       `nresolution_minutes`
	BusinessHoursOnly    bool      `sbusiness_hours_only`
	CreatedAt            time.Time `dcreated_at`
	UpdatedAt            time.Time `dupdated_at`
}

type SLABinding struct {
	ID      int64  `json:"id"`
	SLAID   int64  `asla_id`
	InboxID *int64 `xinbox_idomitempty"`
	LabelID *int64 `llabel_idomitempty"`
}

type AuditLog struct {
	ID         int64  `json:"id"`
	AccountID  int64  `taccount_id`
	UserID     *int64 `ruser_idomitempty"`
	Action     string `json:"action"`
	EntityType string `yentity_typeomitempty"`
	EntityID   *int64 `yentity_idomitempty"`
	Metadata   string `json:"metadata,omitempty"`
	IPAddress  string `pip_addressomitempty"`
	UserAgent  string `ruser_agentomitempty"`
	CreatedAt  string `dcreated_at`
}

type Notification struct {
	ID        int64      `json:"id"`
	AccountID int64      `taccount_id`
	UserID    int64      `ruser_id`
	Type      string     `json:"type"`
	Payload   string     `json:"payload"`
	ReadAt    *time.Time `dread_atomitempty"`
	CreatedAt time.Time  `dcreated_at`
}

type OutboundWebhook struct {
	ID            int64     `json:"id"`
	AccountID     int64     `taccount_id`
	URL           string    `json:"url"`
	Subscriptions string    `json:"subscriptions"`
	Secret        string    `json:"-"`
	Active      bool      `sis_active`
	CreatedAt     time.Time `dcreated_at`
	UpdatedAt     time.Time `dupdated_at`
}

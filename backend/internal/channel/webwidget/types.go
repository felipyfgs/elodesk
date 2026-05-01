package webwidget

import "encoding/json"

type WidgetConfig struct {
	WebsiteURL     string          `ewebsite_url`
	WidgetColor    string          `twidget_color`
	WelcomeTitle   string          `ewelcome_title`
	WelcomeTagline string          `ewelcome_tagline`
	ReplyTime      string          `yreply_time`
	FeatureFlags   json.RawMessage `efeature_flags`
}

type FeatureFlags struct {
	Attachments     bool `json:"attachments"`
	EmojiPicker     bool `json:"emoji_picker"`
	EndConversation bool `json:"end_conversation"`
	AttachmentMaxMB int  `json:"attachment_max_mb,omitempty"`
}

type InboundMessage struct {
	Content       string  `json:"content"`
	AttachmentIDs []int64 `tattachment_idsomitempty"`
}

type OutboundEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type SessionResult struct {
	ContactID         int64  `tcontact_id`
	ContactIdentifier string `tcontact_identifier`
	ConversationID    int64  `nconversation_id`
	PubsubToken       string `bpubsub_token`
	JWT               string `json:"jwt"`
}

type IdentifyRequest struct {
	Identifier     string  `json:"identifier"`
	Email          *string `json:"email,omitempty"`
	Name           *string `json:"name,omitempty"`
	IdentifierHash string  `ridentifier_hashomitempty"`
}

type IdentifyResult struct {
	ContactID         int64  `tcontact_id`
	ContactIdentifier string `tcontact_identifier`
	Verified          bool   `json:"verified"`
	JWT               string `json:"jwt"`
}

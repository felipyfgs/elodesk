package webwidget

import "encoding/json"

type WidgetConfig struct {
	WebsiteURL     string          `json:"websiteUrl"`
	WidgetColor    string          `json:"widgetColor"`
	WelcomeTitle   string          `json:"welcomeTitle"`
	WelcomeTagline string          `json:"welcomeTagline"`
	ReplyTime      string          `json:"replyTime"`
	FeatureFlags   json.RawMessage `json:"featureFlags"`
}

type FeatureFlags struct {
	Attachments     bool `json:"attachments"`
	EmojiPicker     bool `json:"emoji_picker"`
	EndConversation bool `json:"end_conversation"`
	AttachmentMaxMB int  `json:"attachment_max_mb,omitempty"`
}

type InboundMessage struct {
	Content       string  `json:"content"`
	AttachmentIDs []int64 `json:"attachmentIds,omitempty"`
}

type OutboundEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type SessionResult struct {
	ContactID         int64  `json:"contactId"`
	ContactIdentifier string `json:"contactIdentifier"`
	ConversationID    int64  `json:"conversationId"`
	PubsubToken       string `json:"pubsubToken"`
	JWT               string `json:"jwt"`
}

type IdentifyRequest struct {
	Identifier     string  `json:"identifier"`
	Email          *string `json:"email,omitempty"`
	Name           *string `json:"name,omitempty"`
	IdentifierHash string  `json:"identifierHash,omitempty"`
}

type IdentifyResult struct {
	ContactID         int64  `json:"contactId"`
	ContactIdentifier string `json:"contactIdentifier"`
	Verified          bool   `json:"verified"`
	JWT               string `json:"jwt"`
}

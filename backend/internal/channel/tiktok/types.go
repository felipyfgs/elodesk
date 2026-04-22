package tiktok

const (
	APIBase           = "https://business-api.tiktok.com/open_api/v1.3"
	AuthHost          = "https://www.tiktok.com"
	AuthorizePath     = "/v2/auth/authorize"
	TokenEndpoint     = "/tt_user/oauth2/token/"
	RefreshEndpoint   = "/tt_user/oauth2/refresh_token/"
	SendMessageEndpoint = "/business/message/send/"
	BusinessGetEndpoint = "/business/get/"

	EventSendMsg    = "im_send_msg"
	EventReceiveMsg = "im_receive_msg"
	EventMarkRead   = "im_mark_read_msg"

	MessageTypeText  = "text"
	MessageTypeImage = "image"
)

// RequiredScopes are the TikTok Business API scopes required for messaging.
// Kept in sync with _refs/chatwoot/app/services/tiktok/auth_client.rb.
var RequiredScopes = []string{
	"user.info.basic",
	"user.info.username",
	"user.info.stats",
	"user.info.profile",
	"user.account.type",
	"user.insights",
	"message.list.read",
	"message.list.send",
	"message.list.manage",
}

// TokenResponse wraps the access/refresh token envelope returned by
// /tt_user/oauth2/token/ and /tt_user/oauth2/refresh_token/.
type TokenResponse struct {
	Code    int       `json:"code"`
	Message string    `json:"message"`
	Data    TokenData `json:"data"`
}

type TokenData struct {
	OpenID                string `json:"open_id"`
	Scope                 string `json:"scope"`
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	ExpiresIn             int64  `json:"expires_in"`
	RefreshTokenExpiresIn int64  `json:"refresh_token_expires_in"`
}

type BusinessAccountResponse struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Data    BusinessAccountData `json:"data"`
}

type BusinessAccountData struct {
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	ProfileImage string `json:"profile_image"`
}

type SendMessageRequest struct {
	BusinessID    string        `json:"business_id"`
	RecipientType string        `json:"recipient_type"`
	Recipient     string        `json:"recipient"`
	MessageType   string        `json:"message_type"`
	Text          *TextBody     `json:"text,omitempty"`
	Image         *ImagePayload `json:"image,omitempty"`
}

type TextBody struct {
	Body string `json:"body"`
}

type ImagePayload struct {
	MediaID string `json:"media_id"`
}

type SendMessageResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    SendMessageData `json:"data"`
}

type SendMessageData struct {
	Message SendMessageInfo `json:"message"`
}

type SendMessageInfo struct {
	MessageID string `json:"message_id"`
}

// WebhookEvent is the outer envelope from TikTok webhook callbacks.
// `content` is delivered as a JSON-encoded string to be parsed into EventContent.
type WebhookEvent struct {
	Event      string `json:"event"`
	Timestamp  int64  `json:"timestamp"`
	UserOpenID string `json:"user_openid"`
	Content    string `json:"content"`
}

type EventContent struct {
	ConversationID string          `json:"conversation_id"`
	MessageID      string          `json:"message_id"`
	Timestamp      int64           `json:"timestamp"`
	Type           string          `json:"type"`
	Text           *EventTextBody  `json:"text,omitempty"`
	Image          *EventImageBody `json:"image,omitempty"`
	From           string          `json:"from"`
	To             string          `json:"to"`
	FromUser       *EventUser      `json:"from_user,omitempty"`
	ToUser         *EventUser      `json:"to_user,omitempty"`
	Referenced     *Referenced     `json:"referenced_message_info,omitempty"`
}

type EventTextBody struct {
	Body string `json:"body"`
}

type EventImageBody struct {
	MediaID string `json:"media_id"`
}

type EventUser struct {
	ID       string `json:"id"`
	Username string `json:"username,omitempty"`
	Avatar   string `json:"avatar,omitempty"`
}

type Referenced struct {
	ReferencedMessageID string `json:"referenced_message_id"`
}

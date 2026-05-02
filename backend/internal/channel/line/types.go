package line

// WebhookPayload is the root envelope LINE sends on the webhook endpoint.
// https://developers.line.biz/en/reference/messaging-api/#webhook-event-objects
type WebhookPayload struct {
	Destination string  `json:"destination"`
	Events      []Event `json:"events"`
}

type Event struct {
	Type            string           `json:"type"`
	Mode            string           `json:"mode,omitempty"`
	Timestamp       int64            `json:"timestamp"`
	Source          EventSource      `json:"source"`
	WebhookEventID  string           `json:"webhookEventId,omitempty"`
	ReplyToken      string           `json:"replyToken,omitempty"`
	Message         *EventMessage    `json:"message,omitempty"`
	Postback        *EventPostback   `json:"postback,omitempty"`
	DeliveryContext *DeliveryContext `json:"deliveryContext,omitempty"`
}

type DeliveryContext struct {
	IsRedelivery bool `json:"isRedelivery"`
}

type EventSource struct {
	Type    string `json:"type"`
	UserID  string `json:"userId,omitempty"`
	GroupID string `json:"groupId,omitempty"`
	RoomID  string `json:"roomId,omitempty"`
}

type EventMessage struct {
	ID              string            `json:"id"`
	Type            string            `json:"type"`
	Text            string            `json:"text,omitempty"`
	PackageID       string            `json:"packageId,omitempty"`
	StickerID       string            `json:"stickerId,omitempty"`
	FileName        string            `json:"fileName,omitempty"`
	FileSize        int64             `json:"fileSize,omitempty"`
	Title           string            `json:"title,omitempty"`
	Address         string            `json:"address,omitempty"`
	Latitude        float64           `json:"latitude,omitempty"`
	Longitude       float64           `json:"longitude,omitempty"`
	Duration        int64             `json:"duration,omitempty"`
	ContentProvider *ContentProvider  `json:"contentProvider,omitempty"`
	ContentMeta     map[string]any    `json:"contentMeta,omitempty"`
}

type ContentProvider struct {
	Type               string `json:"type"`
	OriginalContentURL string `json:"originalContentUrl,omitempty"`
	PreviewImageURL    string `json:"previewImageUrl,omitempty"`
}

type EventPostback struct {
	Data string `json:"data"`
}

// LINE Messaging API client request/response payloads.

type BotInfo struct {
	UserID          string `json:"userId"`
	BasicID         string `json:"basicId"`
	PremiumID       string `json:"premiumId,omitempty"`
	DisplayName     string `json:"displayName"`
	PictureURL      string `json:"pictureUrl,omitempty"`
	ChatMode        string `json:"chatMode,omitempty"`
	MarkAsReadMode  string `json:"markAsReadMode,omitempty"`
}

type UserProfile struct {
	UserID        string `json:"userId"`
	DisplayName   string `json:"displayName"`
	PictureURL    string `json:"pictureUrl,omitempty"`
	StatusMessage string `json:"statusMessage,omitempty"`
	Language      string `json:"language,omitempty"`
}

type Message struct {
	Type               string `json:"type"`
	Text               string `json:"text,omitempty"`
	OriginalContentURL string `json:"originalContentUrl,omitempty"`
	PreviewImageURL    string `json:"previewImageUrl,omitempty"`
}

type ReplyRequest struct {
	ReplyToken string    `json:"replyToken"`
	Messages   []Message `json:"messages"`
}

type PushRequest struct {
	To       string    `json:"to"`
	Messages []Message `json:"messages"`
}

type APIErrorResponse struct {
	Message string         `json:"message"`
	Details []APIErrorItem `json:"details,omitempty"`
}

type APIErrorItem struct {
	Property string `json:"property"`
	Message  string `json:"message"`
}

// Message content types we recognise from LINE webhook events.
const (
	EventTypeMessage   = "message"
	EventTypeFollow    = "follow"
	EventTypeUnfollow  = "unfollow"
	EventTypePostback  = "postback"
	MessageTypeText    = "text"
	MessageTypeImage   = "image"
	MessageTypeVideo   = "video"
	MessageTypeAudio   = "audio"
	MessageTypeFile    = "file"
	MessageTypeSticker = "sticker"
	MessageTypeLocation = "location"
)

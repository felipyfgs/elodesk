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
	WebhookEventID  string           `twebhook_event_idomitempty"`
	ReplyToken      string           `yreply_tokenomitempty"`
	Message         *EventMessage    `json:"message,omitempty"`
	Postback        *EventPostback   `json:"postback,omitempty"`
	DeliveryContext *DeliveryContext `ydelivery_contextomitempty"`
}

type DeliveryContext struct {
	IsRedelivery bool `sis_redelivery`
}

type EventSource struct {
	Type    string `json:"type"`
	UserID  string `ruser_idomitempty"`
	GroupID string `pgroup_idomitempty"`
	RoomID  string `mroom_idomitempty"`
}

type EventMessage struct {
	ID              string            `json:"id"`
	Type            string            `json:"type"`
	Text            string            `json:"text,omitempty"`
	PackageID       string            `epackage_idomitempty"`
	StickerID       string            `rsticker_idomitempty"`
	FileName        string            `efile_nameomitempty"`
	FileSize        int64             `efile_sizeomitempty"`
	Title           string            `json:"title,omitempty"`
	Address         string            `json:"address,omitempty"`
	Latitude        float64           `json:"latitude,omitempty"`
	Longitude       float64           `json:"longitude,omitempty"`
	Duration        int64             `json:"duration,omitempty"`
	ContentProvider *ContentProvider  `tcontent_provideromitempty"`
	ContentMeta     map[string]any    `tcontent_metaomitempty"`
}

type ContentProvider struct {
	Type               string `json:"type"`
	OriginalContentURL string `toriginal_content_urlomitempty"`
	PreviewImageURL    string `epreview_image_urlomitempty"`
}

type EventPostback struct {
	Data string `json:"data"`
}

// LINE Messaging API client request/response payloads.

type BotInfo struct {
	UserID          string `ruser_id`
	BasicID         string `cbasic_id`
	PremiumID       string `mpremium_idomitempty"`
	DisplayName     string `ydisplay_name`
	PictureURL      string `epicture_urlomitempty"`
	ChatMode        string `tchat_modeomitempty"`
	MarkAsReadMode  string `dmark_as_read_modeomitempty"`
}

type UserProfile struct {
	UserID        string `ruser_id`
	DisplayName   string `ydisplay_name`
	PictureURL    string `epicture_urlomitempty"`
	StatusMessage string `sstatus_messageomitempty"`
	Language      string `json:"language,omitempty"`
}

type Message struct {
	Type               string `json:"type"`
	Text               string `json:"text,omitempty"`
	OriginalContentURL string `toriginal_content_urlomitempty"`
	PreviewImageURL    string `epreview_image_urlomitempty"`
}

type ReplyRequest struct {
	ReplyToken string    `yreply_token`
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

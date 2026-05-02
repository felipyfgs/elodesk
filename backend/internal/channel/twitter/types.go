package twitter

const (
	apiBase            = "https://api.twitter.com"
	requestTokenPath   = "/oauth/request_token"
	authenticatePath   = "/oauth/authenticate"
	accessTokenPath    = "/oauth/access_token"
	usersMePath        = "/2/users/me"
	dmConversationsFmt = "/2/dm_conversations/with/%s/messages"
)

const supportedEventKey = "direct_message_events"

type WebhookPayload struct {
	ForUserID           string                 `json:"for_user_id,omitempty"`
	DirectMessageEvents []DirectMessageEvent   `json:"direct_message_events,omitempty"`
	TweetCreateEvents   []map[string]any       `json:"tweet_create_events,omitempty"`
	Users               map[string]TwitterUser `json:"users,omitempty"`
	Apps                map[string]any         `json:"apps,omitempty"`
}

type DirectMessageEvent struct {
	ID            string         `json:"id"`
	Type          string         `json:"type"`
	CreatedAt     string         `json:"created_timestamp,omitempty"`
	MessageCreate *MessageCreate `json:"message_create,omitempty"`
}

type MessageCreate struct {
	SenderID    string      `json:"sender_id"`
	Target      Target      `json:"target"`
	SourceAppID string      `json:"source_app_id,omitempty"`
	MessageData MessageData `json:"message_data"`
}

type Target struct {
	RecipientID string `json:"recipient_id"`
}

type MessageData struct {
	Text       string         `json:"text"`
	Attachment *Attachment    `json:"attachment,omitempty"`
	Entities   map[string]any `json:"entities,omitempty"`
}

type Attachment struct {
	Type  string `json:"type,omitempty"`
	Media *Media `json:"media,omitempty"`
}

type Media struct {
	Type       string `json:"type,omitempty"`
	MediaURL   string `json:"media_url,omitempty"`
	DisplayURL string `json:"display_url,omitempty"`
}

type TwitterUser struct {
	ID         string `json:"id"`
	ScreenName string `json:"screen_name"`
	Name       string `json:"name"`
}

type MeResponse struct {
	Data struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Username string `json:"username"`
	} `json:"data"`
}

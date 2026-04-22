package twitter

const (
	apiBase            = "https://api.twitter.com"
	requestTokenPath   = "/oauth/request_token"
	authenticatePath   = "/oauth/authenticate"
	accessTokenPath    = "/oauth/access_token"
	usersMePath        = "/2/users/me"
	dmConversationsFmt = "/2/dm_conversations/with/%s/messages"
)

// SupportedEventKey identifies the top-level field of a Twitter Account
// Activity webhook payload that the ingest worker accepts. Anything not in
// this set (e.g. tweet_create_events) is ignored.
const supportedEventKey = "direct_message_events"

// WebhookPayload is the top-level envelope Twitter sends on Account
// Activity webhooks. The fields are intentionally a subset — we only model
// what the DM ingest path consumes.
type WebhookPayload struct {
	ForUserID            string                 `json:"for_user_id,omitempty"`
	DirectMessageEvents  []DirectMessageEvent   `json:"direct_message_events,omitempty"`
	TweetCreateEvents    []map[string]any       `json:"tweet_create_events,omitempty"`
	Users                map[string]TwitterUser `json:"users,omitempty"`
	Apps                 map[string]any         `json:"apps,omitempty"`
}

// DirectMessageEvent is the per-message envelope inside a webhook payload.
// https://developer.twitter.com/en/docs/twitter-api/premium/account-activity-api/guides/account-activity-data-objects
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

// MeResponse is the relevant subset of GET /2/users/me used to resolve the
// authenticated profile id after the OAuth handshake completes.
type MeResponse struct {
	Data struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Username string `json:"username"`
	} `json:"data"`
}

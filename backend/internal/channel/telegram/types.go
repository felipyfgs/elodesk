package telegram

type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	EditedMessage *Message       `json:"edited_message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

type Message struct {
	MessageID      int64      `json:"message_id"`
	From           *User      `json:"from,omitempty"`
	Chat           Chat       `json:"chat"`
	Date           int64      `json:"date"`
	Text           *string    `json:"text,omitempty"`
	Photo          []Photo    `json:"photo,omitempty"`
	Document       *Document  `json:"document,omitempty"`
	Sticker        *Sticker   `json:"sticker,omitempty"`
	Voice          *Voice     `json:"voice,omitempty"`
	Video          *Video     `json:"video,omitempty"`
	VideoNote      *VideoNote `json:"video_note,omitempty"`
	Audio          *Audio     `json:"audio,omitempty"`
	Location       *Location  `json:"location,omitempty"`
	Contact        *Contact   `json:"contact,omitempty"`
	Animation      *Animation `json:"animation,omitempty"`
	Caption        *string    `json:"caption,omitempty"`
	ReplyToMessage *Message   `json:"reply_to_message,omitempty"`
}

type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

type Chat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title,omitempty"`
}

type Photo struct {
	FileID   string `json:"file_id"`
	FileUID  string `json:"file_unique_id"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	FileSize int64  `json:"file_size,omitempty"`
}

type Document struct {
	FileID   string `json:"file_id"`
	FileUID  string `json:"file_unique_id"`
	FileName string `json:"file_name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
}

type Sticker struct {
	FileID     string `json:"file_id"`
	FileUID    string `json:"file_unique_id"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	IsAnimated bool   `json:"is_animated,omitempty"`
	Thumb      *Photo `json:"thumb,omitempty"`
	FileSize   int64  `json:"file_size,omitempty"`
}

type Voice struct {
	FileID   string `json:"file_id"`
	FileUID  string `json:"file_unique_id"`
	Duration int    `json:"duration"`
	MimeType string `json:"mime_type,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
}

type Video struct {
	FileID   string `json:"file_id"`
	FileUID  string `json:"file_unique_id"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Duration int    `json:"duration"`
	MimeType string `json:"mime_type,omitempty"`
	Thumb    *Photo `json:"thumb,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
}

type VideoNote struct {
	FileID   string `json:"file_id"`
	FileUID  string `json:"file_unique_id"`
	Length   int    `json:"length"`
	Duration int    `json:"duration"`
	Thumb    *Photo `json:"thumb,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
}

type Audio struct {
	FileID   string `json:"file_id"`
	FileUID  string `json:"file_unique_id"`
	Duration int    `json:"duration"`
	MimeType string `json:"mime_type,omitempty"`
	Title    string `json:"title,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
}

type Location struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}

type Contact struct {
	PhoneNumber string `json:"phone_number"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name,omitempty"`
	UserID      int64  `json:"user_id,omitempty"`
}

type Animation struct {
	FileID   string `json:"file_id"`
	FileUID  string `json:"file_unique_id"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Duration int    `json:"duration"`
	MimeType string `json:"mime_type,omitempty"`
	Thumb    *Photo `json:"thumb,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data,omitempty"`
}

type APIResponse[T any] struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
	Result      T      `json:"result,omitempty"`
}

type GetMeResult struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username  string `json:"username,omitempty"`
}

type WebhookInfo struct {
	URL                  string `json:"url"`
	HasCustomCertificate bool   `json:"has_custom_certificate"`
	PendingUpdateCount   int    `json:"pending_update_count"`
}

type GetFileResult struct {
	FileID   string `json:"file_id"`
	FileUID  string `json:"file_unique_id"`
	FileSize int64  `json:"file_size"`
	FilePath string `json:"file_path"`
}

type SendMessageRequest struct {
	ChatID           int64        `json:"chat_id"`
	Text             string       `json:"text,omitempty"`
	ParseMode        string       `json:"parse_mode,omitempty"`
	ReplyToMessageID int64        `json:"reply_to_message_id,omitempty"`
	ReplyMarkup      *ReplyMarkup `json:"reply_markup,omitempty"`
}

type SendPhotoRequest struct {
	ChatID           int64        `json:"chat_id"`
	Photo            string       `json:"photo"`
	Caption          string       `json:"caption,omitempty"`
	ParseMode        string       `json:"parse_mode,omitempty"`
	ReplyToMessageID int64        `json:"reply_to_message_id,omitempty"`
	ReplyMarkup      *ReplyMarkup `json:"reply_markup,omitempty"`
}

type SendDocumentRequest struct {
	ChatID           int64        `json:"chat_id"`
	Document         string       `json:"document"`
	Caption          string       `json:"caption,omitempty"`
	ParseMode        string       `json:"parse_mode,omitempty"`
	ReplyToMessageID int64        `json:"reply_to_message_id,omitempty"`
	ReplyMarkup      *ReplyMarkup `json:"reply_markup,omitempty"`
}

type ReplyMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard,omitempty"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"`
}

type MessageSentResult struct {
	MessageID int64 `json:"message_id"`
}

type SetWebhookRequest struct {
	URL            string `json:"url"`
	SecretToken    string `json:"secret_token,omitempty"`
	MaxConnections int    `json:"max_connections,omitempty"`
}

type ButtonDef struct {
	Text         string `json:"text"`
	CallbackData string `json:"callbackData,omitempty"`
	URL          string `json:"url,omitempty"`
}

package meta

import "encoding/json"

// WebhookPayload is the top-level Meta webhook body.
type WebhookPayload struct {
	Object string  `json:"object"`
	Entry  []Entry `json:"entry"`
}

// Entry is one entry in a Meta webhook payload.
type Entry struct {
	ID        string           `json:"id"`
	Time      int64            `json:"time"`
	Messaging []MessagingEntry `json:"messaging"`
	Standby   []MessagingEntry `json:"standby"`
	Changes   []Change         `json:"changes"`
}

// MessagingEntry represents a single messaging event inside an Entry.
type MessagingEntry struct {
	Sender    IDHolder  `json:"sender"`
	Recipient IDHolder  `json:"recipient"`
	Timestamp int64     `json:"timestamp"`
	Message   *Message  `json:"message,omitempty"`
	Delivery  *Delivery `json:"delivery,omitempty"`
	Read      *Read     `json:"read,omitempty"`
	Postback  *Postback `json:"postback,omitempty"`
}

type IDHolder struct {
	ID string `json:"id"`
}

// Message is the message object inside a MessagingEntry.
type Message struct {
	Mid         string       `json:"mid"`
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	IsEcho      bool         `json:"is_echo,omitempty"`
	ReplyTo     *ReplyTo     `json:"reply_to,omitempty"`
	QuickReply  *QuickReply  `json:"quick_reply,omitempty"`
}

type Attachment struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type ReplyTo struct {
	Mid string `json:"mid"`
}

type QuickReply struct {
	Payload string `json:"payload"`
}

type Delivery struct {
	Watermark int64    `json:"watermark"`
	Mids      []string `json:"mids,omitempty"`
}

type Read struct {
	Watermark int64 `json:"watermark"`
}

type Postback struct {
	Title   string `json:"title"`
	Payload string `json:"payload"`
}

type Change struct {
	Field string          `json:"field"`
	Value json.RawMessage `json:"value"`
}

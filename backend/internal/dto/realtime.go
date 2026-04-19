package dto

type RealtimeEvent struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type ConversationCreatedPayload struct {
	ID             int64 `json:"id"`
	AccountID      int64 `json:"accountId"`
	InboxID        int64 `json:"inboxId"`
	ContactID      int64 `json:"contactId"`
	DisplayID      int64 `json:"displayId"`
	ContactInboxID int64 `json:"contactInboxId,omitempty"`
}

type MessageCreatedPayload struct {
	ID             int64  `json:"id"`
	ConversationID int64  `json:"conversationId"`
	MessageType    int    `json:"messageType"`
	ContentType    int    `json:"contentType"`
	Content        string `json:"content,omitempty"`
	Private        bool   `json:"private"`
}

type ConversationStatusChangedPayload struct {
	ID     int64 `json:"id"`
	Status int   `json:"status"`
}

type ConversationAssigneeChangedPayload struct {
	ID         int64  `json:"id"`
	AssigneeID *int64 `json:"assigneeId,omitempty"`
}

type ContactCreatedPayload struct {
	ID        int64  `json:"id"`
	AccountID int64  `json:"accountId"`
	Name      string `json:"name"`
	Email     string `json:"email,omitempty"`
}

type ConversationContactAttributesChangedPayload struct {
	ID        int64 `json:"id"`
	ContactID int64 `json:"contactId"`
}

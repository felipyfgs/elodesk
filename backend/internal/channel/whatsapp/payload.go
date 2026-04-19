package whatsapp

import "encoding/json"

type MetaPayload struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Field string    `json:"field"`
			Value MetaValue `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

type MetaValue struct {
	MessagingProduct string        `json:"messaging_product"`
	Metadata         MetaMetadata  `json:"metadata"`
	Contacts         []MetaContact `json:"contacts"`
	Messages         []MetaMessage `json:"messages"`
	Statuses         []MetaStatus  `json:"statuses"`
	SMBMessageEchoes []MetaMessage `json:"smb_message_echoes"`
}

type MetaMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type MetaContact struct {
	WaID  string `json:"wa_id"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type MetaMessage struct {
	From      string `json:"from"`
	To        string `json:"to"`
	ID        string `json:"id"`
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
	Text      struct {
		Body       string `json:"body"`
		PreviewURL bool   `json:"preview_url"`
	} `json:"text"`
	raw json.RawMessage
}

func (m *MetaMessage) UnmarshalJSON(data []byte) error {
	type Alias MetaMessage
	aux := (*Alias)(m)
	aux.raw = make(json.RawMessage, len(data))
	copy(aux.raw, data)
	return json.Unmarshal(data, aux)
}

type MetaStatus struct {
	ID          string      `json:"id"`
	Status      string      `json:"status"`
	Timestamp   string      `json:"timestamp"`
	RecipientID string      `json:"recipient_id"`
	Errors      []MetaError `json:"errors,omitempty"`
}

type MetaError struct {
	Code    int    `json:"code"`
	Title   string `json:"title"`
	Message string `json:"message,omitempty"`
}

type MetaTemplatesResponse struct {
	Data   []MetaTemplate `json:"data"`
	Paging *MetaPaging    `json:"paging,omitempty"`
}

type MetaTemplate struct {
	Name     string `json:"name"`
	Language string `json:"language"`
	Status   string `json:"status"`
}

type MetaPaging struct {
	Next string `json:"next"`
}

type Dialog360Payload struct {
	Object   string            `json:"object"`
	Messages []Dialog360Msg    `json:"messages"`
	Statuses []Dialog360Status `json:"statuses"`
}

type Dialog360Msg struct {
	From      string        `json:"from"`
	To        string        `json:"to"`
	ID        string        `json:"id"`
	Type      string        `json:"type"`
	Timestamp int64         `json:"timestamp"`
	Text      Dialog360Text `json:"text,omitempty"`
	raw       json.RawMessage
}

func (m *Dialog360Msg) UnmarshalJSON(data []byte) error {
	type Alias Dialog360Msg
	aux := (*Alias)(m)
	aux.raw = make(json.RawMessage, len(data))
	copy(aux.raw, data)
	return json.Unmarshal(data, aux)
}

type Dialog360Text struct {
	Body string `json:"body"`
}

type Dialog360Status struct {
	ID          string           `json:"id"`
	Status      string           `json:"status"`
	Timestamp   string           `json:"timestamp"`
	RecipientID string           `json:"recipient_id"`
	Errors      []Dialog360Error `json:"errors,omitempty"`
}

type Dialog360Error struct {
	Code    int    `json:"code"`
	Title   string `json:"title"`
	Message string `json:"message,omitempty"`
}

type Dialog360TemplatesResponse struct {
	WabaTemplates []Dialog360Template `json:"waba_templates"`
}

type Dialog360Template struct {
	Name     string `json:"name"`
	Language string `json:"language"`
	Status   string `json:"status"`
}

type SendResponse struct {
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

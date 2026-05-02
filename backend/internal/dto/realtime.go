package dto

type RealtimeEvent struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

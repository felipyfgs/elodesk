package dto

// RealtimeEvent is the envelope for every server-sent WebSocket event.
type RealtimeEvent struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// Realtime payloads use the same DTOs as the REST API so the frontend can
// apply them directly to its store without additional fetches.
//
// Event → Payload mapping:
//
//	message.created       → MessageResp (sent by MessageService.broadcastMessageEvent)
//	message.updated       → MessageResp
//	message.deleted       → MessageResp
//	conversation.created  → ConversationResp (full shape, same as GET /:id)
//	conversation.updated  → ConversationResp
//	contact.updated       → ContactResp
//
// Deprecated minimal payload structs below remain for documentation purposes
// only — all realtime emitters now use the full DTOs directly.

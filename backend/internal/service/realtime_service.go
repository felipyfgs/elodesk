package service

import (
	"encoding/json"

	"backend/internal/dto"
	"backend/internal/realtime"
)

// Broadcast safety: `payload` is serialised to JSON verbatim. Domain models
// that carry secrets (model.User.PasswordHash, model.ChannelAPI.HmacToken,
// model.ChannelAPI.Secret, model.ChannelAPI.ApiTokenHash) are tagged json:"-"
// so passing them directly is safe. Callers SHOULD prefer dedicated dto.*
// response types for broadcasts to keep the wire shape explicit.

type RealtimeService struct {
	hub *realtime.Hub
}

func NewRealtimeService(hub *realtime.Hub) *RealtimeService {
	return &RealtimeService{hub: hub}
}

func (s *RealtimeService) BroadcastAccountEvent(accountID int64, eventType string, payload any) {
	msg := dto.RealtimeEvent{
		Type:    eventType,
		Payload: payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	s.hub.Broadcast(realtime.AccountRoom(accountID), data)
}

func (s *RealtimeService) BroadcastInboxEvent(inboxID int64, eventType string, payload any) {
	msg := dto.RealtimeEvent{
		Type:    eventType,
		Payload: payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	s.hub.Broadcast(realtime.InboxRoom(inboxID), data)
}

func (s *RealtimeService) BroadcastConversationEvent(conversationID int64, eventType string, payload any) {
	msg := dto.RealtimeEvent{
		Type:    eventType,
		Payload: payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	s.hub.Broadcast(realtime.ConversationRoom(conversationID), data)
}

package service

import (
	"encoding/json"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/realtime"
)

type RealtimeService struct {
	hub *realtime.Hub
}

func NewRealtimeService(hub *realtime.Hub) *RealtimeService {
	return &RealtimeService{hub: hub}
}

func marshalEvent(event string, payload any) ([]byte, bool) {
	data, err := json.Marshal(dto.RealtimeEvent{Type: event, Payload: payload})
	if err != nil {
		logger.Warn().Str("component", "realtime").Str("event", event).Err(err).Msg("marshal realtime event")
		return nil, false
	}
	return data, true
}

func (s *RealtimeService) BroadcastInboxEvent(inboxID int64, eventType string, payload any) {
	data, ok := marshalEvent(eventType, payload)
	if !ok {
		return
	}
	s.hub.Broadcast(realtime.InboxRoom(inboxID), data)
}

func (s *RealtimeService) BroadcastConversationEvent(conversationID int64, eventType string, payload any) {
	data, ok := marshalEvent(eventType, payload)
	if !ok {
		return
	}
	s.hub.Broadcast(realtime.ConversationRoom(conversationID), data)
}

func (s *RealtimeService) Broadcast(conversationID, accountID int64, event string, payload any) {
	data, ok := marshalEvent(event, payload)
	if !ok {
		return
	}
	logger.Debug().Str("component", "realtime").Str("event", event).
		Int("payload_bytes", len(data)).
		Int64("conversation_id", conversationID).
		Int64("account_id", accountID).
		Msg("broadcast realtime event")
	if conversationID != 0 {
		s.hub.Broadcast(realtime.ConversationRoom(conversationID), data)
	}
	if accountID != 0 {
		s.hub.Broadcast(realtime.AccountRoom(accountID), data)
	}
}

func (s *RealtimeService) BroadcastAccountEvent(accountID int64, eventType string, payload any) {
	data, ok := marshalEvent(eventType, payload)
	if !ok {
		return
	}
	logger.Debug().Str("component", "realtime").Str("event", eventType).
		Int("payload_bytes", len(data)).
		Int64("account_id", accountID).
		Msg("broadcast account realtime event")
	s.hub.Broadcast(realtime.AccountRoom(accountID), data)
}

func (s *RealtimeService) BroadcastUserEvent(accountID, userID int64, eventType string, payload any) {
	data, ok := marshalEvent(eventType, payload)
	if !ok {
		return
	}
	s.hub.Broadcast(realtime.UserRoom(accountID, userID), data)
}

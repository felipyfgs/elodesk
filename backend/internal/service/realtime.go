package service

import (
	"encoding/json"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/realtime"
)

// Broadcast safety: `payload` is serialised to JSON verbatim. Domain models
// that carry secrets (model.User.PasswordHash, model.ChannelAPI.HmacToken,
// model.ChannelAPI.Secret, model.ChannelAPI.ApiTokenHash) are tagged json:"-"
// so passing them directly is safe. Callers SHOULD prefer dedicated dto.*
// response types for broadcasts to keep the wire shape explicit.
//
// Event naming follows `resource.action` (see realtime package constants).
// `conversation.updated` covers status, assignee, team and metadata changes
// — there is no separate event per sub-field, clients diff the payload to
// decide what to re-render.

type RealtimeService struct {
	hub *realtime.Hub
}

func NewRealtimeService(hub *realtime.Hub) *RealtimeService {
	return &RealtimeService{hub: hub}
}

// marshalEvent é o ponto único de serialização para todos os emissores. Mantém
// a forma `{type, payload}` consistente entre broadcasts a salas e
// notificações por usuário.
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

// Broadcast emits `event` with `payload` to both the conversation and account
// rooms. Used by MessageService as the single emission point for message.*
// events so the payload goes to agents viewing the thread (conversation:<id>)
// and to the dashboard/list (account:<aid>). The marshal happens once and the
// resulting bytes are reused across rooms.
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

// BroadcastUserEvent emits to the per-user room (account:N:user:M). Used for
// notifications and any other event scoped to a single agent within a tenant.
func (s *RealtimeService) BroadcastUserEvent(accountID, userID int64, eventType string, payload any) {
	data, ok := marshalEvent(eventType, payload)
	if !ok {
		return
	}
	s.hub.Broadcast(realtime.UserRoom(accountID, userID), data)
}

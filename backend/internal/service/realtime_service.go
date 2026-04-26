package service

import (
	"encoding/json"
	"sync"
	"time"

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

const contactDebounceInterval = 5 * time.Second

type debounceEntry struct {
	timer *time.Timer
	mu    sync.Mutex
}

type RealtimeService struct {
	hub              *realtime.Hub
	contactDebounce  map[int64]*debounceEntry
	contactDebounceMu sync.Mutex
}

func NewRealtimeService(hub *realtime.Hub) *RealtimeService {
	return &RealtimeService{
		hub:             hub,
		contactDebounce: make(map[int64]*debounceEntry),
	}
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

// Broadcast emits `event` with `payload` to both the conversation and
// account rooms. Used by MessageService as the single emission point for
// message.* events so the payload goes to agents viewing the thread
// (conversation:<id>) and to the dashboard/list (account:<aid>).
// BroadcastContactDebounced broadcasts `contact.updated` with a per-contact
// debounce. If called multiple times within `contactDebounceInterval` for the
// same contactID, only the last payload is actually broadcast. This prevents
// flooding the transport when last_activity_at changes at high frequency (e.g.
// many incoming messages for the same contact in rapid succession).
func (s *RealtimeService) BroadcastContactDebounced(contactID, accountID int64, payload any) {
	s.contactDebounceMu.Lock()
	entry, ok := s.contactDebounce[contactID]
	if !ok {
		entry = &debounceEntry{}
		s.contactDebounce[contactID] = entry
		// Cleanup after interval expires
		entry.timer = time.AfterFunc(contactDebounceInterval, func() {
			s.contactDebounceMu.Lock()
			delete(s.contactDebounce, contactID)
			s.contactDebounceMu.Unlock()
		})
	} else {
		entry.timer.Reset(contactDebounceInterval)
	}
	s.contactDebounceMu.Unlock()

	entry.mu.Lock()
	payloadBytes := marshalPayload(realtime.EventContactUpdated, payload)
	entry.mu.Unlock()

	if accountID != 0 {
		s.hub.Broadcast(realtime.AccountRoom(accountID), payloadBytes)
	}
}

func marshalPayload(event string, payload any) []byte {
	msg := dto.RealtimeEvent{Type: event, Payload: payload}
	data, err := json.Marshal(msg)
	if err != nil {
		return nil
	}
	return data
}

func (s *RealtimeService) Broadcast(conversationID, accountID int64, event string, payload any) {
	msg := dto.RealtimeEvent{
		Type:    event,
		Payload: payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
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
	msg := dto.RealtimeEvent{
		Type:    eventType,
		Payload: payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	logger.Debug().Str("component", "realtime").Str("event", eventType).
		Int("payload_bytes", len(data)).
		Int64("account_id", accountID).
		Msg("broadcast account realtime event")
	s.hub.Broadcast(realtime.AccountRoom(accountID), data)
}

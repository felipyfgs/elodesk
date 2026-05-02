package email

import (
	"sync"
	"time"
)

const oauthPendingTTL = 10 * time.Minute

type PendingState struct {
	AccountID int64
	InboxName string
	Provider  string
	ExpiresAt time.Time
}

type OAuthPendingStore struct {
	mu    sync.Mutex
	store map[string]PendingState
}

var GlobalOAuthPending = &OAuthPendingStore{store: make(map[string]PendingState)}

func (s *OAuthPendingStore) Set(state string, p PendingState) {
	p.ExpiresAt = time.Now().Add(oauthPendingTTL)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[state] = p
	s.sweep()
}

func (s *OAuthPendingStore) Get(state string) (PendingState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.store[state]
	if !ok || time.Now().After(p.ExpiresAt) {
		delete(s.store, state)
		return PendingState{}, false
	}
	delete(s.store, state)
	return p, true
}

func (s *OAuthPendingStore) sweep() {
	now := time.Now()
	for k, v := range s.store {
		if now.After(v.ExpiresAt) {
			delete(s.store, k)
		}
	}
}

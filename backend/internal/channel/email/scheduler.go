package email

import (
	"context"
	"fmt"
	"sync"
	"time"

	"backend/internal/logger"
	"backend/internal/model"
)

// Scheduler manages one goroutine per active email channel.
type Scheduler struct {
	mu       sync.Mutex
	pollers  map[int64]context.CancelFunc
	deps     PollDeps
	interval time.Duration
}

func NewScheduler(deps PollDeps, interval time.Duration) *Scheduler {
	return &Scheduler{
		pollers:  make(map[int64]context.CancelFunc),
		deps:     deps,
		interval: interval,
	}
}

// Start spawns a poller goroutine for ch if one is not already running.
func (s *Scheduler) Start(ctx context.Context, ch model.ChannelEmail) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.pollers[ch.ID]; ok {
		return
	}

	pollCtx, cancel := context.WithCancel(ctx)
	s.pollers[ch.ID] = cancel

	deps := s.deps
	deps.InboxID = ch.ID // will be overridden by caller if needed

	poller := NewEmailPoller(ch, deps, s.interval)
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.pollers, ch.ID)
			s.mu.Unlock()
		}()
		logger.Info().Str("component", "email-scheduler").Int64("channelID", ch.ID).Msg("email poller started")
		poller.Run(pollCtx)
		logger.Info().Str("component", "email-scheduler").Int64("channelID", ch.ID).Msg("email poller stopped")
	}()
}

// Stop cancels the poller for channelID (no-op if not running).
func (s *Scheduler) Stop(channelID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cancel, ok := s.pollers[channelID]; ok {
		cancel()
		delete(s.pollers, channelID)
	}
}

// StopAll cancels every running poller.
func (s *Scheduler) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, cancel := range s.pollers {
		cancel()
		delete(s.pollers, id)
		logger.Info().Str("component", "email-scheduler").Int64("channelID", id).Msg("email poller cancelled")
	}
}

// PollerKey returns the reauth tracker key for a channel.
func PollerKey(channelID int64) string {
	return fmt.Sprintf("channel:email:%d", channelID)
}

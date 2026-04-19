package channel

import (
	"fmt"
	"sync"

	"backend/internal/logger"
)

type Registry struct {
	mu    sync.RWMutex
	chans map[Kind]Channel
}

func NewRegistry() *Registry {
	return &Registry{
		chans: make(map[Kind]Channel),
	}
}

func (r *Registry) Register(kind Kind, c Channel) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.chans[kind]; exists {
		logger.Warn().Str("component", "channel.registry").
			Str("kind", string(kind)).
			Msg("duplicate channel registration, last one wins")
	}
	r.chans[kind] = c
}

func (r *Registry) Get(kind Kind) (Channel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.chans[kind]
	if !ok {
		return nil, fmt.Errorf("unknown channel kind")
	}
	return c, nil
}

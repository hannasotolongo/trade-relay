package store

import (
	"errors"
	"sync"

	"github.com/hannasotolongo/trade-relay/internal/trading"
)

var ErrSignalAlreadyExists = errors.New("signal already exists")

type SignalStore struct {
	mu      sync.RWMutex
	signals map[string]trading.Signal
}

func NewSignalStore() *SignalStore {
	return &SignalStore{
		signals: make(map[string]trading.Signal),
	}
}

func (s *SignalStore) Create(signal trading.Signal) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.signals[signal.ID]; exists {
		return ErrSignalAlreadyExists
	}

	s.signals[signal.ID] = signal
	return nil
}

func (s *SignalStore) Get(id string) (trading.Signal, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	signal, exists := s.signals[id]
	return signal, exists
}

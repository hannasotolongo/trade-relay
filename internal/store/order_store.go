package store

import (
	"errors"
	"sync"

	"github.com/hannasotolongo/trade-relay/internal/trading"
)

var (
	ErrOrderAlreadyExists = errors.New("order already exists")
	ErrOrderNotFound      = errors.New("order not found")
)

type OrderStore struct {
	mu     sync.RWMutex
	orders map[string]trading.Order
}

func NewOrderStore() *OrderStore {
	return &OrderStore{
		orders: make(map[string]trading.Order),
	}
}

func (s *OrderStore) Create(order trading.Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.orders[order.ID]; exists {
		return ErrOrderAlreadyExists
	}

	s.orders[order.ID] = order
	return nil
}

func (s *OrderStore) Get(id string) (trading.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	order, exists := s.orders[id]
	if !exists {
		return trading.Order{}, ErrOrderNotFound
	}

	return order, nil
}

func (s *OrderStore) Update(order trading.Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.orders[order.ID]; !exists {
		return ErrOrderNotFound
	}

	s.orders[order.ID] = order
	return nil
}

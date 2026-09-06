package execution

import (
	"errors"
	"testing"
	"time"

	"github.com/hannasotolongo/trade-relay/internal/allocation"
	"github.com/hannasotolongo/trade-relay/internal/trading"
)

func TestNewOrderCreatesOrderFromAllocation(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	alloc := allocation.Allocation{
		AccountID: "acct-001",
		Symbol:    "AAPL",
		Side:      trading.SideBuy,
		Quantity:  25,
	}

	order, err := NewOrder(
		"order-001",
		"signal-001",
		"broker-acct-001",
		alloc,
		now,
	)
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	if order.ID != "order-001" {
		t.Fatalf("expected order ID %q, got %q", "order-001", order.ID)
	}

	if order.SignalID != "signal-001" {
		t.Fatalf("expected signal ID %q, got %q", "signal-001", order.SignalID)
	}

	if order.AccountID != "acct-001" {
		t.Fatalf("expected account ID %q, got %q", "acct-001", order.AccountID)
	}

	if order.BrokerAccountID != "broker-acct-001" {
		t.Fatalf("expected broker account ID %q, got %q", "broker-acct-001", order.BrokerAccountID)
	}

	if order.Symbol != "AAPL" {
		t.Fatalf("expected symbol AAPL, got %q", order.Symbol)
	}

	if order.Side != trading.SideBuy {
		t.Fatalf("expected side %s, got %s", trading.SideBuy, order.Side)
	}

	if order.Quantity != 25 {
		t.Fatalf("expected quantity 25, got %d", order.Quantity)
	}

	if order.FilledQuantity != 0 {
		t.Fatalf("expected filled quantity 0, got %d", order.FilledQuantity)
	}

	if order.Status != trading.OrderCreated {
		t.Fatalf(
			"expected status %s, got %s",
			trading.OrderCreated,
			order.Status,
		)
	}

	if !order.CreatedAt.Equal(now) {
		t.Fatalf("expected CreatedAt %v, got %v", now, order.CreatedAt)
	}

	if !order.UpdatedAt.Equal(now) {
		t.Fatalf("expected UpdatedAt %v, got %v", now, order.UpdatedAt)
	}
}

func TestNewOrderRejectsMissingOrderID(t *testing.T) {
	alloc := allocation.Allocation{
		AccountID: "acct-001",
		Symbol:    "AAPL",
		Side:      trading.SideBuy,
		Quantity:  25,
	}

	_, err := NewOrder(
		"",
		"signal-001",
		"broker-acct-001",
		alloc,
		time.Now(),
	)

	if !errors.Is(err, ErrMissingOrderID) {
		t.Fatalf("expected ErrMissingOrderID, got %v", err)
	}
}

func TestNewOrderRejectsInvalidQuantity(t *testing.T) {
	alloc := allocation.Allocation{
		AccountID: "acct-001",
		Symbol:    "AAPL",
		Side:      trading.SideBuy,
		Quantity:  0,
	}

	_, err := NewOrder(
		"order-001",
		"signal-001",
		"broker-acct-001",
		alloc,
		time.Now(),
	)

	if !errors.Is(err, trading.ErrInvalidQuantity) {
		t.Fatalf("expected ErrInvalidQuantity, got %v", err)
	}
}

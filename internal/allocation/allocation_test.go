package allocation

import (
	"errors"
	"testing"
	"time"

	"github.com/hannasotolongo/trade-relay/internal/trading"
)

func testSignal(quantity int64) trading.Signal {
	return trading.Signal{
		ID:         "sig-001",
		StrategyID: "strategy-001",
		Symbol:     "AAPL",
		Side:       trading.SideBuy,
		Quantity:   quantity,
		CreatedAt:  time.Now(),
	}
}

func testAccount(allocationBps int64) trading.Account {
	return trading.Account{
		ID:               "acct-001",
		BrokerAccountID:  "broker-001",
		Status:           trading.AccountActive,
		CashBalanceCents: 1_000_000,
		AllocationBps:    allocationBps,
		MaxPositionCents: 300_000,
	}
}

func TestCalculateAllocatesQuantity(t *testing.T) {
	signal := testSignal(100)
	account := testAccount(2500)

	got, err := Calculate(signal, account)
	if err != nil {
		t.Fatalf("calculate allocation: %v", err)
	}

	if got.Quantity != 25 {
		t.Fatalf("expected quantity 25, got %d", got.Quantity)
	}

	if got.AccountID != account.ID {
		t.Fatalf("expected account %q, got %q", account.ID, got.AccountID)
	}

	if got.Symbol != signal.Symbol {
		t.Fatalf("expected symbol %q, got %q", signal.Symbol, got.Symbol)
	}

	if got.Side != signal.Side {
		t.Fatalf("expected side %q, got %q", signal.Side, got.Side)
	}
}

func TestCalculateRejectsDisabledAccount(t *testing.T) {
	signal := testSignal(100)
	account := testAccount(2500)
	account.Status = trading.AccountDisabled

	_, err := Calculate(signal, account)
	if !errors.Is(err, ErrAccountNotActive) {
		t.Fatalf("expected ErrAccountNotActive, got %v", err)
	}
}

func TestCalculateRejectsAllocationTooSmall(t *testing.T) {
	signal := testSignal(3)
	account := testAccount(2500)

	_, err := Calculate(signal, account)
	if !errors.Is(err, ErrAllocationTooSmall) {
		t.Fatalf("expected ErrAllocationTooSmall, got %v", err)
	}
}

func TestCalculateRejectsInvalidSignal(t *testing.T) {
	signal := testSignal(0)
	account := testAccount(2500)

	_, err := Calculate(signal, account)
	if !errors.Is(err, trading.ErrInvalidQuantity) {
		t.Fatalf("expected ErrInvalidQuantity, got %v", err)
	}
}

func TestCalculateRejectsInvalidAccount(t *testing.T) {
	signal := testSignal(100)
	account := testAccount(10001)

	_, err := Calculate(signal, account)
	if !errors.Is(err, trading.ErrInvalidAllocationBps) {
		t.Fatalf("expected ErrInvalidAllocationBps, got %v", err)
	}
}
func TestCalculateManyHandlesPartialSuccess(t *testing.T) {
	signal := testSignal(100)

	accounts := []trading.Account{
		{
			ID:               "acct-001",
			BrokerAccountID:  "broker-001",
			Status:           trading.AccountActive,
			CashBalanceCents: 1_000_000,
			AllocationBps:    5000,
			MaxPositionCents: 300_000,
		},
		{
			ID:               "acct-002",
			BrokerAccountID:  "broker-002",
			Status:           trading.AccountDisabled,
			CashBalanceCents: 1_000_000,
			AllocationBps:    2500,
			MaxPositionCents: 300_000,
		},
		{
			ID:               "acct-003",
			BrokerAccountID:  "broker-003",
			Status:           trading.AccountActive,
			CashBalanceCents: 1_000_000,
			AllocationBps:    1000,
			MaxPositionCents: 300_000,
		},
	}

	result := CalculateMany(signal, accounts)

	if len(result.Allocations) != 2 {
		t.Fatalf("expected 2 successful allocations, got %d", len(result.Allocations))
	}

	if len(result.Failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(result.Failures))
	}

	if result.Allocations[0].AccountID != "acct-001" {
		t.Fatalf("expected first allocation for acct-001, got %q", result.Allocations[0].AccountID)
	}

	if result.Allocations[0].Quantity != 50 {
		t.Fatalf("expected acct-001 quantity 50, got %d", result.Allocations[0].Quantity)
	}

	if result.Allocations[1].AccountID != "acct-003" {
		t.Fatalf("expected second allocation for acct-003, got %q", result.Allocations[1].AccountID)
	}

	if result.Allocations[1].Quantity != 10 {
		t.Fatalf("expected acct-003 quantity 10, got %d", result.Allocations[1].Quantity)
	}

	if result.Failures[0].AccountID != "acct-002" {
		t.Fatalf("expected failure for acct-002, got %q", result.Failures[0].AccountID)
	}

	if !errors.Is(result.Failures[0].Err, ErrAccountNotActive) {
		t.Fatalf("expected ErrAccountNotActive, got %v", result.Failures[0].Err)
	}
}

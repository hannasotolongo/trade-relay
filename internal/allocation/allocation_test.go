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

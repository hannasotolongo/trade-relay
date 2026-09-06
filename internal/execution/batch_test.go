package execution

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hannasotolongo/trade-relay/internal/allocation"
	"github.com/hannasotolongo/trade-relay/internal/trading"
)

type fakeAccountLookup struct {
	accounts map[string]trading.Account
}

func (f fakeAccountLookup) GetAccount(accountID string) (trading.Account, bool) {
	account, exists := f.accounts[accountID]
	return account, exists
}

type batchOrderStore struct {
	orders map[string]trading.Order
}

func newBatchOrderStore() *batchOrderStore {
	return &batchOrderStore{
		orders: make(map[string]trading.Order),
	}
}

func (s *batchOrderStore) Create(order trading.Order) error {
	if _, exists := s.orders[order.ID]; exists {
		return errors.New("duplicate order")
	}

	s.orders[order.ID] = order
	return nil
}

func (s *batchOrderStore) Update(order trading.Order) error {
	if _, exists := s.orders[order.ID]; !exists {
		return errors.New("order not found")
	}

	s.orders[order.ID] = order
	return nil
}

func TestExecuteAllocationsExecutesMultipleAccounts(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	accounts := fakeAccountLookup{
		accounts: map[string]trading.Account{
			"acct-001": {
				ID:              "acct-001",
				BrokerAccountID: "broker-001",
			},
			"acct-002": {
				ID:              "acct-002",
				BrokerAccountID: "broker-002",
			},
		},
	}

	allocations := []allocation.Allocation{
		{
			AccountID: "acct-001",
			Symbol:    "AAPL",
			Side:      trading.SideBuy,
			Quantity:  50,
		},
		{
			AccountID: "acct-002",
			Symbol:    "AAPL",
			Side:      trading.SideBuy,
			Quantity:  25,
		},
	}

	store := newBatchOrderStore()

	coordinator := Coordinator{
		Broker: successfulBatchBroker{},
		Store:  store,
	}

	result := ExecuteAllocations(
		context.Background(),
		coordinator,
		accounts,
		"signal-001",
		allocations,
		func(signalID, accountID string) string {
			return signalID + "-" + accountID
		},
		now,
	)

	if len(result.Orders) != 2 {
		t.Fatalf("expected 2 orders, got %d", len(result.Orders))
	}

	if len(result.Failures) != 0 {
		t.Fatalf("expected no failures, got %d", len(result.Failures))
	}

	for _, order := range result.Orders {
		if order.Status != trading.OrderAcknowledged {
			t.Fatalf(
				"expected order %s to be acknowledged, got %s",
				order.ID,
				order.Status,
			)
		}
	}
}

func TestExecuteAllocationsContinuesAfterMissingAccount(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	accounts := fakeAccountLookup{
		accounts: map[string]trading.Account{
			"acct-001": {
				ID:              "acct-001",
				BrokerAccountID: "broker-001",
			},
		},
	}

	allocations := []allocation.Allocation{
		{
			AccountID: "missing-account",
			Symbol:    "AAPL",
			Side:      trading.SideBuy,
			Quantity:  10,
		},
		{
			AccountID: "acct-001",
			Symbol:    "AAPL",
			Side:      trading.SideBuy,
			Quantity:  20,
		},
	}

	store := newBatchOrderStore()

	coordinator := Coordinator{
		Broker: successfulBatchBroker{},
		Store:  store,
	}

	result := ExecuteAllocations(
		context.Background(),
		coordinator,
		accounts,
		"signal-001",
		allocations,
		func(signalID, accountID string) string {
			return signalID + "-" + accountID
		},
		now,
	)

	if len(result.Orders) != 1 {
		t.Fatalf("expected 1 successful order, got %d", len(result.Orders))
	}

	if len(result.Failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(result.Failures))
	}

	if result.Failures[0].AccountID != "missing-account" {
		t.Fatalf(
			"expected failure for missing-account, got %s",
			result.Failures[0].AccountID,
		)
	}

	if result.Orders[0].AccountID != "acct-001" {
		t.Fatalf(
			"expected acct-001 to continue executing, got %s",
			result.Orders[0].AccountID,
		)
	}
}

type successfulBatchBroker struct{}

func (successfulBatchBroker) SubmitOrder(
	ctx context.Context,
	order trading.Order,
) (trading.BrokerResult, error) {
	if err := ctx.Err(); err != nil {
		return trading.BrokerResult{}, err
	}

	return trading.BrokerResult{
		OrderID: "broker-" + order.ID,
		Status:  trading.OrderAcknowledged,
	}, nil
}
func (successfulBatchBroker) GetOrder(
	ctx context.Context,
	brokerOrderID string,
) (trading.BrokerResult, error) {
	if err := ctx.Err(); err != nil {
		return trading.BrokerResult{}, err
	}

	return trading.BrokerResult{
		OrderID: brokerOrderID,
		Status:  trading.OrderAcknowledged,
	}, nil
}
func (successfulBatchBroker) GetOrderByClientID(
	ctx context.Context,
	clientOrderID string,
) (trading.BrokerResult, error) {
	if err := ctx.Err(); err != nil {
		return trading.BrokerResult{}, err
	}

	return trading.BrokerResult{
		OrderID: "broker-" + clientOrderID,
		Status:  trading.OrderAcknowledged,
	}, nil
}

package execution

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hannasotolongo/trade-relay/internal/broker"
	"github.com/hannasotolongo/trade-relay/internal/trading"
)

func TestReconcileUnknownToPartiallyFilled(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	order := trading.Order{
		ID:             "order-001",
		BrokerOrderID:  "broker-order-001",
		Quantity:       100,
		FilledQuantity: 0,
		Status:         trading.OrderUnknown,
	}

	store := &fakeOrderStore{}

	reconciler := Reconciler{
		Broker: broker.SimulatedBroker{
			GetResult: trading.BrokerResult{
				OrderID:        "broker-order-001",
				Status:         trading.OrderPartiallyFilled,
				FilledQuantity: 40,
			},
		},
		Store: store,
	}

	err := reconciler.Reconcile(
		context.Background(),
		&order,
		now,
	)
	if err != nil {
		t.Fatalf("reconcile order: %v", err)
	}

	if order.Status != trading.OrderPartiallyFilled {
		t.Fatalf(
			"expected status %s, got %s",
			trading.OrderPartiallyFilled,
			order.Status,
		)
	}

	if order.FilledQuantity != 40 {
		t.Fatalf(
			"expected filled quantity 40, got %d",
			order.FilledQuantity,
		)
	}

	if len(store.updates) != 1 {
		t.Fatalf("expected 1 store update, got %d", len(store.updates))
	}

	if store.updates[0].Status != trading.OrderPartiallyFilled {
		t.Fatalf(
			"expected stored status %s, got %s",
			trading.OrderPartiallyFilled,
			store.updates[0].Status,
		)
	}
}

func TestReconcilePartiallyFilledToFilled(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 5, 0, 0, time.UTC)

	order := trading.Order{
		ID:             "order-001",
		BrokerOrderID:  "broker-order-001",
		Quantity:       100,
		FilledQuantity: 40,
		Status:         trading.OrderPartiallyFilled,
	}

	store := &fakeOrderStore{}

	reconciler := Reconciler{
		Broker: broker.SimulatedBroker{
			GetResult: trading.BrokerResult{
				OrderID:        "broker-order-001",
				Status:         trading.OrderFilled,
				FilledQuantity: 100,
			},
		},
		Store: store,
	}

	err := reconciler.Reconcile(
		context.Background(),
		&order,
		now,
	)
	if err != nil {
		t.Fatalf("reconcile order: %v", err)
	}

	if order.Status != trading.OrderFilled {
		t.Fatalf(
			"expected status %s, got %s",
			trading.OrderFilled,
			order.Status,
		)
	}

	if order.FilledQuantity != 100 {
		t.Fatalf(
			"expected filled quantity 100, got %d",
			order.FilledQuantity,
		)
	}
}

func TestReconcileRejectsMissingBrokerOrderID(t *testing.T) {
	order := trading.Order{
		ID:       "order-001",
		Quantity: 100,
		Status:   trading.OrderUnknown,
	}

	reconciler := Reconciler{
		Broker: broker.SimulatedBroker{},
		Store:  &fakeOrderStore{},
	}

	err := reconciler.Reconcile(
		context.Background(),
		&order,
		time.Now(),
	)

	if !errors.Is(err, ErrMissingBrokerOrderID) {
		t.Fatalf(
			"expected ErrMissingBrokerOrderID, got %v",
			err,
		)
	}
}

func TestReconcileRejectsUnexpectedBrokerOrderID(t *testing.T) {
	order := trading.Order{
		ID:            "order-001",
		BrokerOrderID: "broker-order-001",
		Quantity:      100,
		Status:        trading.OrderUnknown,
	}

	reconciler := Reconciler{
		Broker: broker.SimulatedBroker{
			GetResult: trading.BrokerResult{
				OrderID:        "different-order",
				Status:         trading.OrderFilled,
				FilledQuantity: 100,
			},
		},
		Store: &fakeOrderStore{},
	}

	err := reconciler.Reconcile(
		context.Background(),
		&order,
		time.Now(),
	)

	if err == nil {
		t.Fatal("expected reconciliation error")
	}
}

func TestReconcileRejectsInvalidFilledQuantity(t *testing.T) {
	order := trading.Order{
		ID:            "order-001",
		BrokerOrderID: "broker-order-001",
		Quantity:      100,
		Status:        trading.OrderUnknown,
	}

	reconciler := Reconciler{
		Broker: broker.SimulatedBroker{
			GetResult: trading.BrokerResult{
				OrderID:        "broker-order-001",
				Status:         trading.OrderFilled,
				FilledQuantity: 150,
			},
		},
		Store: &fakeOrderStore{},
	}

	err := reconciler.Reconcile(
		context.Background(),
		&order,
		time.Now(),
	)

	if !errors.Is(err, ErrInvalidFilledQuantity) {
		t.Fatalf(
			"expected ErrInvalidFilledQuantity, got %v",
			err,
		)
	}
}

func TestReconcileReturnsBrokerError(t *testing.T) {
	brokerErr := errors.New("broker lookup failed")

	order := trading.Order{
		ID:            "order-001",
		BrokerOrderID: "broker-order-001",
		Quantity:      100,
		Status:        trading.OrderUnknown,
	}

	reconciler := Reconciler{
		Broker: broker.SimulatedBroker{
			GetErr: brokerErr,
		},
		Store: &fakeOrderStore{},
	}

	err := reconciler.Reconcile(
		context.Background(),
		&order,
		time.Now(),
	)

	if !errors.Is(err, brokerErr) {
		t.Fatalf("expected broker error, got %v", err)
	}
}

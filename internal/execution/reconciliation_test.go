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

func TestReconcileRejectsMissingOrderReference(t *testing.T) {
	order := trading.Order{
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

	if !errors.Is(err, ErrMissingOrderReference) {
		t.Fatalf(
			"expected %v, got %v",
			ErrMissingOrderReference,
			err,
		)
	}
}

func TestReconcileUsesClientOrderIDWhenBrokerOrderIDMissing(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 10, 0, 0, time.UTC)

	order := trading.Order{
		ID:             "order-001",
		Quantity:       100,
		FilledQuantity: 0,
		Status:         trading.OrderUnknown,
	}

	store := &fakeOrderStore{}

	reconciler := Reconciler{
		Broker: broker.SimulatedBroker{
			GetByClientResult: trading.BrokerResult{
				OrderID:        "broker-order-001",
				Status:         trading.OrderAcknowledged,
				FilledQuantity: 0,
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

	if order.Status != trading.OrderAcknowledged {
		t.Fatalf(
			"expected status %s, got %s",
			trading.OrderAcknowledged,
			order.Status,
		)
	}

	if order.BrokerOrderID != "broker-order-001" {
		t.Fatalf(
			"expected broker order ID %q, got %q",
			"broker-order-001",
			order.BrokerOrderID,
		)
	}

	if len(store.updates) != 1 {
		t.Fatalf(
			"expected 1 store update, got %d",
			len(store.updates),
		)
	}

	if store.updates[0].BrokerOrderID != "broker-order-001" {
		t.Fatalf(
			"expected stored broker order ID %q, got %q",
			"broker-order-001",
			store.updates[0].BrokerOrderID,
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

	if !errors.Is(err, ErrUnexpectedBrokerOrderID) {
		t.Fatalf(
			"expected %v, got %v",
			ErrUnexpectedBrokerOrderID,
			err,
		)
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
			"expected %v, got %v",
			ErrInvalidFilledQuantity,
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

func TestReconcileReturnsClientLookupError(t *testing.T) {
	brokerErr := errors.New("client order lookup failed")

	order := trading.Order{
		ID:       "order-001",
		Quantity: 100,
		Status:   trading.OrderUnknown,
	}

	reconciler := Reconciler{
		Broker: broker.SimulatedBroker{
			GetByClientErr: brokerErr,
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

func TestReconcileRejectsFilledQuantityRegression(t *testing.T) {
	order := trading.Order{
		ID:             "order-001",
		BrokerOrderID:  "broker-order-001",
		Quantity:       100,
		FilledQuantity: 40,
		Status:         trading.OrderPartiallyFilled,
	}

	reconciler := Reconciler{
		Broker: broker.SimulatedBroker{
			GetResult: trading.BrokerResult{
				OrderID:        "broker-order-001",
				Status:         trading.OrderPartiallyFilled,
				FilledQuantity: 20,
			},
		},
		Store: &fakeOrderStore{},
	}

	err := reconciler.Reconcile(
		context.Background(),
		&order,
		time.Now(),
	)

	if !errors.Is(err, ErrFilledQuantityRegression) {
		t.Fatalf(
			"expected %v, got %v",
			ErrFilledQuantityRegression,
			err,
		)
	}

	if order.FilledQuantity != 40 {
		t.Fatalf(
			"expected filled quantity to remain 40, got %d",
			order.FilledQuantity,
		)
	}

	if order.Status != trading.OrderPartiallyFilled {
		t.Fatalf(
			"expected status to remain %s, got %s",
			trading.OrderPartiallyFilled,
			order.Status,
		)
	}
}

func TestReconcileRejectsFilledStatusWithIncompleteQuantity(t *testing.T) {
	order := trading.Order{
		ID:             "order-001",
		BrokerOrderID:  "broker-order-001",
		Quantity:       100,
		FilledQuantity: 40,
		Status:         trading.OrderPartiallyFilled,
	}

	reconciler := Reconciler{
		Broker: broker.SimulatedBroker{
			GetResult: trading.BrokerResult{
				OrderID:        "broker-order-001",
				Status:         trading.OrderFilled,
				FilledQuantity: 90,
			},
		},
		Store: &fakeOrderStore{},
	}

	err := reconciler.Reconcile(
		context.Background(),
		&order,
		time.Now(),
	)

	if !errors.Is(err, ErrInconsistentBrokerState) {
		t.Fatalf(
			"expected %v, got %v",
			ErrInconsistentBrokerState,
			err,
		)
	}
}

func TestReconcileRejectsPartialFillWithZeroQuantity(t *testing.T) {
	order := trading.Order{
		ID:             "order-001",
		BrokerOrderID:  "broker-order-001",
		Quantity:       100,
		FilledQuantity: 0,
		Status:         trading.OrderAcknowledged,
	}

	reconciler := Reconciler{
		Broker: broker.SimulatedBroker{
			GetResult: trading.BrokerResult{
				OrderID:        "broker-order-001",
				Status:         trading.OrderPartiallyFilled,
				FilledQuantity: 0,
			},
		},
		Store: &fakeOrderStore{},
	}

	err := reconciler.Reconcile(
		context.Background(),
		&order,
		time.Now(),
	)

	if !errors.Is(err, ErrInconsistentBrokerState) {
		t.Fatalf(
			"expected %v, got %v",
			ErrInconsistentBrokerState,
			err,
		)
	}
}

func TestReconcileRejectsPartialFillAtFullQuantity(t *testing.T) {
	order := trading.Order{
		ID:             "order-001",
		BrokerOrderID:  "broker-order-001",
		Quantity:       100,
		FilledQuantity: 40,
		Status:         trading.OrderPartiallyFilled,
	}

	reconciler := Reconciler{
		Broker: broker.SimulatedBroker{
			GetResult: trading.BrokerResult{
				OrderID:        "broker-order-001",
				Status:         trading.OrderPartiallyFilled,
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

	if !errors.Is(err, ErrInconsistentBrokerState) {
		t.Fatalf(
			"expected %v, got %v",
			ErrInconsistentBrokerState,
			err,
		)
	}
}

func TestReconcileRejectsAcknowledgedOrderWithFill(t *testing.T) {
	order := trading.Order{
		ID:             "order-001",
		BrokerOrderID:  "broker-order-001",
		Quantity:       100,
		FilledQuantity: 0,
		Status:         trading.OrderUnknown,
	}

	reconciler := Reconciler{
		Broker: broker.SimulatedBroker{
			GetResult: trading.BrokerResult{
				OrderID:        "broker-order-001",
				Status:         trading.OrderAcknowledged,
				FilledQuantity: 10,
			},
		},
		Store: &fakeOrderStore{},
	}

	err := reconciler.Reconcile(
		context.Background(),
		&order,
		time.Now(),
	)

	if !errors.Is(err, ErrInconsistentBrokerState) {
		t.Fatalf(
			"expected %v, got %v",
			ErrInconsistentBrokerState,
			err,
		)
	}
}

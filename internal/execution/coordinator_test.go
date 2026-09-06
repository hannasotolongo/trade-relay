package execution

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hannasotolongo/trade-relay/internal/broker"
	"github.com/hannasotolongo/trade-relay/internal/trading"
)

type fakeOrderStore struct {
	creates []trading.Order
	updates []trading.Order
	err     error
}

func (s *fakeOrderStore) Create(order trading.Order) error {
	if s.err != nil {
		return s.err
	}

	s.creates = append(s.creates, order)
	return nil
}

func (s *fakeOrderStore) Update(order trading.Order) error {
	if s.err != nil {
		return s.err
	}

	s.updates = append(s.updates, order)
	return nil
}

func validCoordinatorOrder() trading.Order {
	return trading.Order{
		ID:              "order-001",
		SignalID:        "signal-001",
		AccountID:       "account-001",
		BrokerAccountID: "broker-account-001",
		Symbol:          "AAPL",
		Side:            trading.SideBuy,
		Quantity:        100,
		FilledQuantity:  0,
		Status:          trading.OrderCreated,
	}
}

func TestSubmitAcknowledgesOrder(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	order := validCoordinatorOrder()
	store := &fakeOrderStore{}

	coordinator := Coordinator{
		Broker: &broker.SimulatedBroker{
			Result: trading.BrokerResult{
				OrderID: "broker-order-001",
				Status:  trading.OrderAcknowledged,
			},
		},
		Store: store,
	}

	err := coordinator.Submit(
		context.Background(),
		&order,
		now,
	)
	if err != nil {
		t.Fatalf("submit order: %v", err)
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

	if len(store.updates) != 2 {
		t.Fatalf("expected 2 store updates, got %d", len(store.updates))
	}

	if store.updates[0].Status != trading.OrderSubmitting {
		t.Fatalf(
			"expected first stored status %s, got %s",
			trading.OrderSubmitting,
			store.updates[0].Status,
		)
	}

	if store.updates[1].Status != trading.OrderAcknowledged {
		t.Fatalf(
			"expected second stored status %s, got %s",
			trading.OrderAcknowledged,
			store.updates[1].Status,
		)
	}

	if store.updates[1].BrokerOrderID != "broker-order-001" {
		t.Fatalf(
			"expected stored broker order ID %q, got %q",
			"broker-order-001",
			store.updates[1].BrokerOrderID,
		)
	}
}

func TestSubmitMarksOrderUnknownOnBrokerError(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	brokerErr := errors.New("broker unavailable")

	order := validCoordinatorOrder()
	store := &fakeOrderStore{}

	coordinator := Coordinator{
		Broker: &broker.SimulatedBroker{
			Err: brokerErr,
		},
		Store: store,
	}

	err := coordinator.Submit(
		context.Background(),
		&order,
		now,
	)

	if !errors.Is(err, brokerErr) {
		t.Fatalf("expected broker error, got %v", err)
	}

	if order.Status != trading.OrderUnknown {
		t.Fatalf(
			"expected status %s, got %s",
			trading.OrderUnknown,
			order.Status,
		)
	}

	if order.BrokerOrderID != "" {
		t.Fatalf(
			"expected broker order ID to remain empty, got %q",
			order.BrokerOrderID,
		)
	}

	if len(store.updates) != 2 {
		t.Fatalf("expected 2 store updates, got %d", len(store.updates))
	}

	if store.updates[1].Status != trading.OrderUnknown {
		t.Fatalf(
			"expected final stored status %s, got %s",
			trading.OrderUnknown,
			store.updates[1].Status,
		)
	}
}

func TestSubmitRejectsInvalidStartingState(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	order := validCoordinatorOrder()
	order.Status = trading.OrderFilled
	order.FilledQuantity = order.Quantity

	store := &fakeOrderStore{}

	coordinator := Coordinator{
		Broker: &broker.SimulatedBroker{},
		Store:  store,
	}

	err := coordinator.Submit(
		context.Background(),
		&order,
		now,
	)

	if !errors.Is(err, trading.ErrInvalidOrderTransition) {
		t.Fatalf(
			"expected ErrInvalidOrderTransition, got %v",
			err,
		)
	}

	if order.Status != trading.OrderFilled {
		t.Fatalf(
			"expected status to remain %s, got %s",
			trading.OrderFilled,
			order.Status,
		)
	}

	if len(store.updates) != 0 {
		t.Fatalf("expected no store updates, got %d", len(store.updates))
	}
}

func TestSubmitMarksOrderUnknownWhenContextCanceled(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	order := validCoordinatorOrder()
	store := &fakeOrderStore{}

	coordinator := Coordinator{
		Broker: &broker.SimulatedBroker{
			Result: trading.BrokerResult{
				OrderID: "broker-order-001",
				Status:  trading.OrderAcknowledged,
			},
		},
		Store: store,
	}

	err := coordinator.Submit(
		ctx,
		&order,
		now,
	)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	if order.Status != trading.OrderUnknown {
		t.Fatalf(
			"expected status %s, got %s",
			trading.OrderUnknown,
			order.Status,
		)
	}

	if order.BrokerOrderID != "" {
		t.Fatalf(
			"expected broker order ID to remain empty, got %q",
			order.BrokerOrderID,
		)
	}

	if len(store.updates) != 2 {
		t.Fatalf("expected 2 store updates, got %d", len(store.updates))
	}
}

func TestSubmitReturnsStoreError(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	storeErr := errors.New("store unavailable")

	order := validCoordinatorOrder()

	store := &fakeOrderStore{
		err: storeErr,
	}

	coordinator := Coordinator{
		Broker: &broker.SimulatedBroker{},
		Store:  store,
	}

	err := coordinator.Submit(
		context.Background(),
		&order,
		now,
	)

	if !errors.Is(err, storeErr) {
		t.Fatalf("expected store error, got %v", err)
	}
}

func TestSubmitMarksOrderRejectedWhenDefinitelyNotSent(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	submitErr := trading.NewBrokerSubmissionError(
		trading.SubmissionNotSent,
		errors.New("validation failed before submission"),
	)

	order := validCoordinatorOrder()
	store := &fakeOrderStore{}

	coordinator := Coordinator{
		Broker: &broker.SimulatedBroker{
			Err: submitErr,
		},
		Store: store,
	}

	err := coordinator.Submit(
		context.Background(),
		&order,
		now,
	)

	if err == nil {
		t.Fatal("expected submission error")
	}

	if order.Status != trading.OrderRejected {
		t.Fatalf(
			"expected status %s, got %s",
			trading.OrderRejected,
			order.Status,
		)
	}

	if len(store.updates) != 2 {
		t.Fatalf("expected 2 store updates, got %d", len(store.updates))
	}

	if store.updates[1].Status != trading.OrderRejected {
		t.Fatalf(
			"expected stored status %s, got %s",
			trading.OrderRejected,
			store.updates[1].Status,
		)
	}
}

func TestSubmitMarksOrderUnknownWhenOutcomeAmbiguous(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	submitErr := trading.NewBrokerSubmissionError(
		trading.SubmissionAmbiguous,
		errors.New("connection lost after request"),
	)

	order := validCoordinatorOrder()
	store := &fakeOrderStore{}

	coordinator := Coordinator{
		Broker: &broker.SimulatedBroker{
			Err: submitErr,
		},
		Store: store,
	}

	err := coordinator.Submit(
		context.Background(),
		&order,
		now,
	)

	if err == nil {
		t.Fatal("expected submission error")
	}

	if order.Status != trading.OrderUnknown {
		t.Fatalf(
			"expected status %s, got %s",
			trading.OrderUnknown,
			order.Status,
		)
	}

	if len(store.updates) != 2 {
		t.Fatalf("expected 2 store updates, got %d", len(store.updates))
	}

	if store.updates[1].Status != trading.OrderUnknown {
		t.Fatalf(
			"expected stored status %s, got %s",
			trading.OrderUnknown,
			store.updates[1].Status,
		)
	}
}

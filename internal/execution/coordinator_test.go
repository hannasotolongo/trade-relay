package execution

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hannasotolongo/trade-relay/internal/broker"
	"github.com/hannasotolongo/trade-relay/internal/trading"
)

func TestSubmitAcknowledgesOrder(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	order := trading.Order{
		ID:     "order-001",
		Status: trading.OrderCreated,
	}

	coordinator := Coordinator{
		Broker: broker.SimulatedBroker{
			Result: trading.BrokerResult{
				OrderID: "broker-order-001",
				Status:  trading.OrderAcknowledged,
			},
		},
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

	if !order.UpdatedAt.Equal(now) {
		t.Fatalf("expected UpdatedAt %v, got %v", now, order.UpdatedAt)
	}
}

func TestSubmitMarksOrderUnknownOnBrokerError(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	brokerErr := errors.New("broker unavailable")

	order := trading.Order{
		ID:     "order-001",
		Status: trading.OrderCreated,
	}

	coordinator := Coordinator{
		Broker: broker.SimulatedBroker{
			Err: brokerErr,
		},
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
}

func TestSubmitRejectsInvalidStartingState(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	order := trading.Order{
		ID:     "order-001",
		Status: trading.OrderFilled,
	}

	coordinator := Coordinator{
		Broker: broker.SimulatedBroker{},
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
}

func TestSubmitMarksOrderUnknownWhenContextCanceled(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	order := trading.Order{
		ID:     "order-001",
		Status: trading.OrderCreated,
	}

	coordinator := Coordinator{
		Broker: broker.SimulatedBroker{
			Result: trading.BrokerResult{
				OrderID: "broker-order-001",
				Status:  trading.OrderAcknowledged,
			},
		},
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
}

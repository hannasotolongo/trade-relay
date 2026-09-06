package broker

import (
	"context"
	"errors"
	"testing"

	"github.com/hannasotolongo/trade-relay/internal/trading"
)

func TestSimulatedBrokerReturnsConfiguredResult(t *testing.T) {
	b := SimulatedBroker{
		Result: trading.BrokerResult{
			OrderID: "broker-order-001",
			Status:  trading.OrderAcknowledged,
		},
	}

	result, err := b.SubmitOrder(
		context.Background(),
		trading.Order{ID: "order-001"},
	)
	if err != nil {
		t.Fatalf("submit order: %v", err)
	}

	if result.OrderID != "broker-order-001" {
		t.Fatalf("expected broker order ID %q, got %q", "broker-order-001", result.OrderID)
	}

	if result.Status != trading.OrderAcknowledged {
		t.Fatalf("expected status %s, got %s", trading.OrderAcknowledged, result.Status)
	}
}

func TestSimulatedBrokerReturnsConfiguredError(t *testing.T) {
	expectedErr := errors.New("broker unavailable")

	b := SimulatedBroker{
		Err: expectedErr,
	}

	_, err := b.SubmitOrder(
		context.Background(),
		trading.Order{ID: "order-001"},
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestSimulatedBrokerRespectsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	b := SimulatedBroker{
		Result: trading.BrokerResult{
			OrderID: "broker-order-001",
			Status:  trading.OrderAcknowledged,
		},
	}

	_, err := b.SubmitOrder(
		ctx,
		trading.Order{ID: "order-001"},
	)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

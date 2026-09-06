package broker

import (
	"context"
	"errors"
	"testing"

	"github.com/hannasotolongo/trade-relay/internal/trading"
)

func TestSimulatedBrokerSubmitOrderReturnsConfiguredResult(t *testing.T) {
	expected := trading.BrokerResult{
		OrderID:        "broker-order-001",
		Status:         trading.OrderAcknowledged,
		FilledQuantity: 0,
	}

	b := SimulatedBroker{
		Result: expected,
	}

	result, err := b.SubmitOrder(
		context.Background(),
		trading.Order{
			ID:     "order-001",
			Status: trading.OrderSubmitting,
		},
	)
	if err != nil {
		t.Fatalf("submit order: %v", err)
	}

	if result != expected {
		t.Fatalf("expected %+v, got %+v", expected, result)
	}
}

func TestSimulatedBrokerSubmitOrderReturnsConfiguredError(t *testing.T) {
	expectedErr := errors.New("submit failed")

	b := SimulatedBroker{
		Err: expectedErr,
	}

	_, err := b.SubmitOrder(
		context.Background(),
		trading.Order{},
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestSimulatedBrokerSubmitOrderRespectsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	b := SimulatedBroker{}

	_, err := b.SubmitOrder(ctx, trading.Order{})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestSimulatedBrokerGetOrderReturnsConfiguredResult(t *testing.T) {
	expected := trading.BrokerResult{
		OrderID:        "broker-order-001",
		Status:         trading.OrderPartiallyFilled,
		FilledQuantity: 40,
	}

	b := SimulatedBroker{
		GetResult: expected,
	}

	result, err := b.GetOrder(
		context.Background(),
		"broker-order-001",
	)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}

	if result != expected {
		t.Fatalf("expected %+v, got %+v", expected, result)
	}
}

func TestSimulatedBrokerGetOrderReturnsConfiguredError(t *testing.T) {
	expectedErr := errors.New("lookup failed")

	b := SimulatedBroker{
		GetErr: expectedErr,
	}

	_, err := b.GetOrder(
		context.Background(),
		"broker-order-001",
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestSimulatedBrokerGetOrderRespectsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	b := SimulatedBroker{}

	_, err := b.GetOrder(
		ctx,
		"broker-order-001",
	)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

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
func TestSubmitOrderIsIdempotentByClientOrderID(t *testing.T) {
	b := &SimulatedBroker{}

	order := trading.Order{
		ID:              "order-001",
		SignalID:        "signal-001",
		AccountID:       "account-001",
		BrokerAccountID: "broker-account-001",
		Symbol:          "AAPL",
		Side:            trading.SideBuy,
		Quantity:        100,
		Status:          trading.OrderCreated,
	}

	first, err := b.SubmitOrder(
		context.Background(),
		order,
	)
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}

	second, err := b.SubmitOrder(
		context.Background(),
		order,
	)
	if err != nil {
		t.Fatalf("second submit: %v", err)
	}

	if first.OrderID != second.OrderID {
		t.Fatalf(
			"expected same broker order ID, got %q and %q",
			first.OrderID,
			second.OrderID,
		)
	}

	if first.OrderID != "broker-order-001" {
		t.Fatalf(
			"expected broker order ID %q, got %q",
			"broker-order-001",
			first.OrderID,
		)
	}
}
func TestSubmitOrderConcurrentIdempotency(t *testing.T) {
	b := &SimulatedBroker{}

	order := trading.Order{
		ID:              "order-001",
		SignalID:        "signal-001",
		AccountID:       "account-001",
		BrokerAccountID: "broker-account-001",
		Symbol:          "AAPL",
		Side:            trading.SideBuy,
		Quantity:        100,
		Status:          trading.OrderCreated,
	}

	const goroutines = 20

	results := make(chan trading.BrokerResult, goroutines)
	errs := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			result, err := b.SubmitOrder(
				context.Background(),
				order,
			)

			results <- result
			errs <- err
		}()
	}

	var expectedOrderID string

	for i := 0; i < goroutines; i++ {
		err := <-errs
		if err != nil {
			t.Fatalf("submit order: %v", err)
		}

		result := <-results

		if expectedOrderID == "" {
			expectedOrderID = result.OrderID
		}

		if result.OrderID != expectedOrderID {
			t.Fatalf(
				"expected broker order ID %q, got %q",
				expectedOrderID,
				result.OrderID,
			)
		}
	}

	if expectedOrderID != "broker-order-001" {
		t.Fatalf(
			"expected broker order ID %q, got %q",
			"broker-order-001",
			expectedOrderID,
		)
	}
}

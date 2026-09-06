package trading

import (
	"errors"
	"testing"
	"time"
)

func TestCanTransitionOrderAllowsValidTransitions(t *testing.T) {
	tests := []struct {
		from OrderStatus
		to   OrderStatus
	}{
		{OrderCreated, OrderSubmitting},
		{OrderSubmitting, OrderAcknowledged},
		{OrderSubmitting, OrderRejected},
		{OrderSubmitting, OrderUnknown},
		{OrderAcknowledged, OrderPartiallyFilled},
		{OrderAcknowledged, OrderFilled},
		{OrderPartiallyFilled, OrderPartiallyFilled},
		{OrderPartiallyFilled, OrderFilled},
		{OrderUnknown, OrderAcknowledged},
		{OrderUnknown, OrderPartiallyFilled},
		{OrderUnknown, OrderFilled},
		{OrderUnknown, OrderRejected},
	}

	for _, tt := range tests {
		if !CanTransitionOrder(tt.from, tt.to) {
			t.Fatalf(
				"expected transition %s -> %s to be allowed",
				tt.from,
				tt.to,
			)
		}
	}
}

func TestCanTransitionOrderRejectsInvalidTransitions(t *testing.T) {
	tests := []struct {
		from OrderStatus
		to   OrderStatus
	}{
		{OrderCreated, OrderFilled},
		{OrderCreated, OrderRejected},
		{OrderFilled, OrderSubmitting},
		{OrderFilled, OrderPartiallyFilled},
		{OrderRejected, OrderAcknowledged},
	}

	for _, tt := range tests {
		if CanTransitionOrder(tt.from, tt.to) {
			t.Fatalf(
				"expected transition %s -> %s to be rejected",
				tt.from,
				tt.to,
			)
		}
	}
}

func TestTransitionOrderUpdatesStatusAndTimestamp(t *testing.T) {
	order := Order{
		Status: OrderCreated,
	}

	now := time.Now()

	err := TransitionOrder(&order, OrderSubmitting, now)
	if err != nil {
		t.Fatalf("transition order: %v", err)
	}

	if order.Status != OrderSubmitting {
		t.Fatalf(
			"expected status %s, got %s",
			OrderSubmitting,
			order.Status,
		)
	}

	if !order.UpdatedAt.Equal(now) {
		t.Fatalf(
			"expected updated time %v, got %v",
			now,
			order.UpdatedAt,
		)
	}
}

func TestTransitionOrderRejectsInvalidTransitionWithoutMutation(t *testing.T) {
	originalTime := time.Now()

	order := Order{
		Status:    OrderCreated,
		UpdatedAt: originalTime,
	}

	err := TransitionOrder(&order, OrderFilled, time.Now())
	if !errors.Is(err, ErrInvalidOrderTransition) {
		t.Fatalf(
			"expected ErrInvalidOrderTransition, got %v",
			err,
		)
	}

	if order.Status != OrderCreated {
		t.Fatalf(
			"expected status to remain %s, got %s",
			OrderCreated,
			order.Status,
		)
	}

	if !order.UpdatedAt.Equal(originalTime) {
		t.Fatalf("expected UpdatedAt to remain unchanged")
	}
}

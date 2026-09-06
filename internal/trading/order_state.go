package trading

import (
	"errors"
	"time"
)

var ErrInvalidOrderTransition = errors.New("invalid order state transition")

func CanTransitionOrder(from, to OrderStatus) bool {
	switch from {
	case OrderCreated:
		return to == OrderSubmitting

	case OrderSubmitting:
		return to == OrderAcknowledged ||
			to == OrderRejected ||
			to == OrderUnknown

	case OrderAcknowledged:
		return to == OrderPartiallyFilled ||
			to == OrderFilled

	case OrderPartiallyFilled:
		return to == OrderPartiallyFilled ||
			to == OrderFilled

	case OrderUnknown:
		return to == OrderAcknowledged ||
			to == OrderPartiallyFilled ||
			to == OrderFilled ||
			to == OrderRejected

	case OrderFilled, OrderRejected:
		return false

	default:
		return false
	}
}

func TransitionOrder(order *Order, to OrderStatus, now time.Time) error {
	if !CanTransitionOrder(order.Status, to) {
		return ErrInvalidOrderTransition
	}

	order.Status = to
	order.UpdatedAt = now

	return nil
}

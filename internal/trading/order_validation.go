package trading

import (
	"errors"
	"strings"
)

var (
	ErrMissingOrderID        = errors.New("missing order ID")
	ErrMissingOrderAccountID = errors.New("missing order account ID")
	ErrMissingOrderBrokerID  = errors.New("missing broker account ID")
	ErrMissingOrderSymbol    = errors.New("missing order symbol")
	ErrInvalidOrderQuantity  = errors.New("invalid order quantity")
	ErrInvalidFilledQuantity = errors.New("invalid filled quantity")
	ErrInvalidOrderSide      = errors.New("invalid order side")
	ErrInvalidOrderStatus    = errors.New("invalid order status")
)

func ValidateOrder(order Order) error {
	if strings.TrimSpace(order.ID) == "" {
		return ErrMissingOrderID
	}

	if strings.TrimSpace(order.SignalID) == "" {
		return ErrMissingSignalID
	}

	if strings.TrimSpace(order.AccountID) == "" {
		return ErrMissingOrderAccountID
	}

	if strings.TrimSpace(order.BrokerAccountID) == "" {
		return ErrMissingOrderBrokerID
	}

	if strings.TrimSpace(order.Symbol) == "" {
		return ErrMissingOrderSymbol
	}

	if order.Side != SideBuy && order.Side != SideSell {
		return ErrInvalidOrderSide
	}

	if order.Quantity <= 0 {
		return ErrInvalidOrderQuantity
	}

	if order.FilledQuantity < 0 ||
		order.FilledQuantity > order.Quantity {
		return ErrInvalidFilledQuantity
	}

	switch order.Status {
	case OrderCreated,
		OrderSubmitting,
		OrderAcknowledged,
		OrderPartiallyFilled,
		OrderFilled,
		OrderRejected,
		OrderUnknown:
		return nil
	default:
		return ErrInvalidOrderStatus
	}
}

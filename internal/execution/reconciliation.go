package execution

import (
	"context"
	"errors"
	"time"

	"github.com/hannasotolongo/trade-relay/internal/trading"
)

var (
	ErrMissingOrderReference    = errors.New("missing order reference")
	ErrInvalidFilledQuantity    = errors.New("invalid filled quantity")
	ErrFilledQuantityRegression = errors.New("filled quantity cannot decrease")
	ErrInconsistentBrokerState  = errors.New("broker status and filled quantity are inconsistent")
	ErrUnexpectedBrokerOrderID  = errors.New("broker returned unexpected order ID")
)

type Reconciler struct {
	Broker trading.Broker
	Store  OrderStore
}

func (r Reconciler) Reconcile(
	ctx context.Context,
	order *trading.Order,
	now time.Time,
) error {
	if order.ID == "" && order.BrokerOrderID == "" {
		return ErrMissingOrderReference
	}

	var (
		result trading.BrokerResult
		err    error
	)

	if order.BrokerOrderID != "" {
		result, err = r.Broker.GetOrder(
			ctx,
			order.BrokerOrderID,
		)
	} else {
		result, err = r.Broker.GetOrderByClientID(
			ctx,
			order.ID,
		)
	}

	if err != nil {
		return err
	}

	if order.BrokerOrderID != "" &&
		result.OrderID != order.BrokerOrderID {
		return ErrUnexpectedBrokerOrderID
	}

	if result.OrderID == "" {
		return ErrUnexpectedBrokerOrderID
	}

	if result.FilledQuantity < 0 ||
		result.FilledQuantity > order.Quantity {
		return ErrInvalidFilledQuantity
	}

	if result.FilledQuantity < order.FilledQuantity {
		return ErrFilledQuantityRegression
	}

	switch result.Status {
	case trading.OrderFilled:
		if result.FilledQuantity != order.Quantity {
			return ErrInconsistentBrokerState
		}

	case trading.OrderPartiallyFilled:
		if result.FilledQuantity <= 0 ||
			result.FilledQuantity >= order.Quantity {
			return ErrInconsistentBrokerState
		}

	case trading.OrderAcknowledged,
		trading.OrderRejected:
		if result.FilledQuantity != 0 {
			return ErrInconsistentBrokerState
		}
	}

	if err := trading.TransitionOrder(
		order,
		result.Status,
		now,
	); err != nil {
		return err
	}

	order.BrokerOrderID = result.OrderID
	order.FilledQuantity = result.FilledQuantity

	return r.Store.Update(order)
}

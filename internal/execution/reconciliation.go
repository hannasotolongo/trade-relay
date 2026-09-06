package execution

import (
	"context"
	"errors"
	"time"

	"github.com/hannasotolongo/trade-relay/internal/trading"
)

var (
	ErrMissingBrokerOrderID     = errors.New("missing broker order ID")
	ErrInvalidFilledQuantity    = errors.New("invalid filled quantity")
	ErrFilledQuantityRegression = errors.New("filled quantity cannot decrease")
	ErrInconsistentBrokerState  = errors.New("broker status and filled quantity are inconsistent")
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
	if order.BrokerOrderID == "" {
		return ErrMissingBrokerOrderID
	}

	result, err := r.Broker.GetOrder(
		ctx,
		order.BrokerOrderID,
	)
	if err != nil {
		return err
	}

	if result.OrderID != order.BrokerOrderID {
		return errors.New("broker returned unexpected order ID")
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

	order.FilledQuantity = result.FilledQuantity

	return r.Store.Update(*order)
}

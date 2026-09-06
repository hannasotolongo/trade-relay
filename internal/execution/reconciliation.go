package execution

import (
	"context"
	"errors"
	"time"

	"github.com/hannasotolongo/trade-relay/internal/trading"
)

var (
	ErrMissingBrokerOrderID  = errors.New("missing broker order ID")
	ErrInvalidFilledQuantity = errors.New("invalid filled quantity")
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

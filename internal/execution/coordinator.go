package execution

import (
	"context"
	"time"

	"github.com/hannasotolongo/trade-relay/internal/trading"
)

type OrderStore interface {
	Update(order trading.Order) error
}

type Coordinator struct {
	Broker trading.Broker
	Store  OrderStore
}

func (c Coordinator) Submit(
	ctx context.Context,
	order *trading.Order,
	now time.Time,
) error {
	if err := trading.TransitionOrder(
		order,
		trading.OrderSubmitting,
		now,
	); err != nil {
		return err
	}

	if err := c.Store.Update(*order); err != nil {
		return err
	}

	result, err := c.Broker.SubmitOrder(ctx, *order)
	if err != nil {
		if transitionErr := trading.TransitionOrder(
			order,
			trading.OrderUnknown,
			now,
		); transitionErr != nil {
			return transitionErr
		}

		if updateErr := c.Store.Update(*order); updateErr != nil {
			return updateErr
		}

		return err
	}

	if err := trading.TransitionOrder(
		order,
		result.Status,
		now,
	); err != nil {
		return err
	}

	return c.Store.Update(*order)
}

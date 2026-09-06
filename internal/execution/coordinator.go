package execution

import (
	"context"
	"time"

	"github.com/hannasotolongo/trade-relay/internal/trading"
)

type Coordinator struct {
	Broker trading.Broker
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

	result, err := c.Broker.SubmitOrder(ctx, *order)
	if err != nil {
		if transitionErr := trading.TransitionOrder(
			order,
			trading.OrderUnknown,
			now,
		); transitionErr != nil {
			return transitionErr
		}

		return err
	}

	return trading.TransitionOrder(
		order,
		result.Status,
		now,
	)
}

package broker

import (
	"context"

	"github.com/hannasotolongo/trade-relay/internal/trading"
)

type SimulatedBroker struct {
	Result trading.BrokerResult
	Err    error
}

func (b SimulatedBroker) SubmitOrder(
	ctx context.Context,
	order trading.Order,
) (trading.BrokerResult, error) {
	if err := ctx.Err(); err != nil {
		return trading.BrokerResult{}, err
	}

	if b.Err != nil {
		return trading.BrokerResult{}, b.Err
	}

	return b.Result, nil
}

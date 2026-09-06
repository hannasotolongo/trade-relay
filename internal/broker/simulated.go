package broker

import (
	"context"

	"github.com/hannasotolongo/trade-relay/internal/trading"
)

type SimulatedBroker struct {
	Result            trading.BrokerResult
	Err               error
	GetResult         trading.BrokerResult
	GetErr            error
	GetByClientResult trading.BrokerResult
	GetByClientErr    error
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

func (b SimulatedBroker) GetOrder(
	ctx context.Context,
	brokerOrderID string,
) (trading.BrokerResult, error) {
	if err := ctx.Err(); err != nil {
		return trading.BrokerResult{}, err
	}

	if b.GetErr != nil {
		return trading.BrokerResult{}, b.GetErr
	}

	return b.GetResult, nil
}

func (b SimulatedBroker) GetOrderByClientID(
	ctx context.Context,
	clientOrderID string,
) (trading.BrokerResult, error) {
	if err := ctx.Err(); err != nil {
		return trading.BrokerResult{}, err
	}

	if b.GetByClientErr != nil {
		return trading.BrokerResult{}, b.GetByClientErr
	}

	return b.GetByClientResult, nil
}

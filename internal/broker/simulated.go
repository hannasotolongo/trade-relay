package broker

import (
	"context"
	"sync"

	"github.com/hannasotolongo/trade-relay/internal/trading"
)

type SimulatedBroker struct {
	mu sync.Mutex

	Result            trading.BrokerResult
	Err               error
	GetResult         trading.BrokerResult
	GetErr            error
	GetByClientResult trading.BrokerResult
	GetByClientErr    error

	ordersByClientID map[string]trading.BrokerResult
}

func (b *SimulatedBroker) SubmitOrder(
	ctx context.Context,
	order trading.Order,
) (trading.BrokerResult, error) {
	if err := ctx.Err(); err != nil {
		return trading.BrokerResult{}, err
	}

	if b.Err != nil {
		return trading.BrokerResult{}, b.Err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.ordersByClientID == nil {
		b.ordersByClientID = make(map[string]trading.BrokerResult)
	}

	if existing, exists := b.ordersByClientID[order.ID]; exists {
		return existing, nil
	}

	result := b.Result

	if result.OrderID == "" {
		result.OrderID = "broker-" + order.ID
	}

	if result.Status == "" {
		result.Status = trading.OrderAcknowledged
	}

	b.ordersByClientID[order.ID] = result

	return result, nil
}

func (b *SimulatedBroker) GetOrder(
	ctx context.Context,
	brokerOrderID string,
) (trading.BrokerResult, error) {
	if err := ctx.Err(); err != nil {
		return trading.BrokerResult{}, err
	}

	if b.GetErr != nil {
		return trading.BrokerResult{}, b.GetErr
	}

	if b.GetResult.OrderID != "" {
		return b.GetResult, nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for _, result := range b.ordersByClientID {
		if result.OrderID == brokerOrderID {
			return result, nil
		}
	}

	return trading.BrokerResult{}, nil
}

func (b *SimulatedBroker) GetOrderByClientID(
	ctx context.Context,
	clientOrderID string,
) (trading.BrokerResult, error) {
	if err := ctx.Err(); err != nil {
		return trading.BrokerResult{}, err
	}

	if b.GetByClientErr != nil {
		return trading.BrokerResult{}, b.GetByClientErr
	}

	if b.GetByClientResult.OrderID != "" {
		return b.GetByClientResult, nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	return b.ordersByClientID[clientOrderID], nil
}

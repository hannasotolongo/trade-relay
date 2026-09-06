package trading

import "context"

type BrokerResult struct {
	OrderID string
	Status  OrderStatus
}

type Broker interface {
	SubmitOrder(ctx context.Context, order Order) (BrokerResult, error)
}

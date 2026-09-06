package trading

import "context"

type BrokerResult struct {
	OrderID        string
	Status         OrderStatus
	FilledQuantity int64
}

type Broker interface {
	SubmitOrder(
		ctx context.Context,
		order Order,
	) (BrokerResult, error)

	GetOrder(
		ctx context.Context,
		brokerOrderID string,
	) (BrokerResult, error)

	GetOrderByClientID(
		ctx context.Context,
		clientOrderID string,
	) (BrokerResult, error)
}

package trading

import "time"

type OrderStatus string

const (
	OrderCreated         OrderStatus = "CREATED"
	OrderSubmitting      OrderStatus = "SUBMITTING"
	OrderAcknowledged    OrderStatus = "ACKNOWLEDGED"
	OrderPartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	OrderFilled          OrderStatus = "FILLED"
	OrderRejected        OrderStatus = "REJECTED"
	OrderUnknown         OrderStatus = "UNKNOWN"
)

type Order struct {
	ID              string
	SignalID        string
	AccountID       string
	BrokerAccountID string
	BrokerOrderID   string
	Symbol          string
	Side            Side
	Quantity        int64
	FilledQuantity  int64
	Status          OrderStatus
	Version         int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

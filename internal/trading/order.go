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
	Symbol          string
	Side            Side
	Quantity        int64
	FilledQuantity  int64
	Status          OrderStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

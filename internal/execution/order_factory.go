package execution

import (
	"errors"
	"time"

	"github.com/hannasotolongo/trade-relay/internal/allocation"
	"github.com/hannasotolongo/trade-relay/internal/trading"
)

var ErrMissingOrderID = errors.New("missing order ID")

func NewOrder(
	orderID string,
	signalID string,
	brokerAccountID string,
	allocation allocation.Allocation,
	now time.Time,
) (trading.Order, error) {
	if orderID == "" {
		return trading.Order{}, ErrMissingOrderID
	}

	if signalID == "" {
		return trading.Order{}, trading.ErrMissingSignalID
	}

	if allocation.AccountID == "" {
		return trading.Order{}, trading.ErrMissingAccountID
	}

	if brokerAccountID == "" {
		return trading.Order{}, trading.ErrMissingBrokerAccountID
	}

	if allocation.Quantity <= 0 {
		return trading.Order{}, trading.ErrInvalidQuantity
	}

	return trading.Order{
		ID:              orderID,
		SignalID:        signalID,
		AccountID:       allocation.AccountID,
		BrokerAccountID: brokerAccountID,
		Symbol:          allocation.Symbol,
		Side:            allocation.Side,
		Quantity:        allocation.Quantity,
		FilledQuantity:  0,
		Status:          trading.OrderCreated,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

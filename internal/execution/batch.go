package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/hannasotolongo/trade-relay/internal/allocation"
	"github.com/hannasotolongo/trade-relay/internal/trading"
)

type AccountLookup interface {
	GetAccount(accountID string) (trading.Account, bool)
}

type BatchFailure struct {
	AccountID string
	Err       error
}

type BatchResult struct {
	Orders   []trading.Order
	Failures []BatchFailure
}

type OrderIDGenerator func(
	signalID string,
	accountID string,
) string

func ExecuteAllocations(
	ctx context.Context,
	coordinator Coordinator,
	accounts AccountLookup,
	signalID string,
	allocations []allocation.Allocation,
	generateOrderID OrderIDGenerator,
	now time.Time,
) BatchResult {
	result := BatchResult{}

	for _, alloc := range allocations {
		account, exists := accounts.GetAccount(alloc.AccountID)
		if !exists {
			result.Failures = append(
				result.Failures,
				BatchFailure{
					AccountID: alloc.AccountID,
					Err:       fmt.Errorf("account not found: %s", alloc.AccountID),
				},
			)
			continue
		}

		order, err := NewOrder(
			generateOrderID(signalID, alloc.AccountID),
			signalID,
			account.BrokerAccountID,
			alloc,
			now,
		)
		if err != nil {
			result.Failures = append(
				result.Failures,
				BatchFailure{
					AccountID: alloc.AccountID,
					Err:       err,
				},
			)
			continue
		}

		if err := coordinator.Store.Create(order); err != nil {
			result.Failures = append(
				result.Failures,
				BatchFailure{
					AccountID: alloc.AccountID,
					Err:       err,
				},
			)
			continue
		}

		if err := coordinator.Submit(ctx, &order, now); err != nil {
			result.Failures = append(
				result.Failures,
				BatchFailure{
					AccountID: alloc.AccountID,
					Err:       err,
				},
			)

			result.Orders = append(result.Orders, order)
			continue
		}

		result.Orders = append(result.Orders, order)
	}

	return result
}

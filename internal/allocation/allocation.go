package allocation

import (
	"errors"

	"github.com/hannasotolongo/trade-relay/internal/trading"
)

var (
	ErrAccountNotActive   = errors.New("account is not active")
	ErrAllocationTooSmall = errors.New("allocation results in zero quantity")
)

type Allocation struct {
	AccountID string
	Symbol    string
	Side      trading.Side
	Quantity  int64
}

func Calculate(signal trading.Signal, account trading.Account) (Allocation, error) {
	if err := trading.ValidateSignal(signal); err != nil {
		return Allocation{}, err
	}

	if err := trading.ValidateAccount(account); err != nil {
		return Allocation{}, err
	}

	if account.Status != trading.AccountActive {
		return Allocation{}, ErrAccountNotActive
	}

	quantity := signal.Quantity * account.AllocationBps / 10000

	if quantity == 0 {
		return Allocation{}, ErrAllocationTooSmall
	}

	return Allocation{
		AccountID: account.ID,
		Symbol:    signal.Symbol,
		Side:      signal.Side,
		Quantity:  quantity,
	}, nil
}

type Failure struct {
	AccountID string
	Err       error
}

type Result struct {
	Allocations []Allocation
	Failures    []Failure
}

func CalculateMany(signal trading.Signal, accounts []trading.Account) Result {
	result := Result{}

	for _, account := range accounts {
		allocation, err := Calculate(signal, account)
		if err != nil {
			result.Failures = append(result.Failures, Failure{
				AccountID: account.ID,
				Err:       err,
			})
			continue
		}

		result.Allocations = append(result.Allocations, allocation)
	}

	return result
}

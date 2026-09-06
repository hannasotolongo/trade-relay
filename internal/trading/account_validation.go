package trading

import (
	"errors"
	"strings"
)

var (
	ErrMissingAccountID       = errors.New("account ID is required")
	ErrMissingBrokerAccountID = errors.New("broker account ID is required")
	ErrInvalidAccountStatus   = errors.New("account status must be ACTIVE or DISABLED")
	ErrInvalidCashBalance     = errors.New("cash balance cannot be negative")
	ErrInvalidAllocationBps   = errors.New("allocation basis points must be between 0 and 10000")
	ErrInvalidMaxPosition     = errors.New("max position cannot be negative")
)

func ValidateAccount(account Account) error {
	if strings.TrimSpace(account.ID) == "" {
		return ErrMissingAccountID
	}

	if strings.TrimSpace(account.BrokerAccountID) == "" {
		return ErrMissingBrokerAccountID
	}

	if account.Status != AccountActive && account.Status != AccountDisabled {
		return ErrInvalidAccountStatus
	}

	if account.CashBalanceCents < 0 {
		return ErrInvalidCashBalance
	}

	if account.AllocationBps < 0 || account.AllocationBps > 10000 {
		return ErrInvalidAllocationBps
	}

	if account.MaxPositionCents < 0 {
		return ErrInvalidMaxPosition
	}

	return nil
}

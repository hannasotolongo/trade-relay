package trading

import (
	"errors"
	"testing"
)

func validAccount() Account {
	return Account{
		ID:               "acct-001",
		BrokerAccountID:  "broker-001",
		Status:           AccountActive,
		CashBalanceCents: 1_000_000,
		AllocationBps:    2500,
		MaxPositionCents: 300_000,
	}
}

func TestValidateAccountAcceptsValidAccount(t *testing.T) {
	account := validAccount()

	if err := ValidateAccount(account); err != nil {
		t.Fatalf("expected valid account, got error: %v", err)
	}
}

func TestValidateAccountRejectsMissingID(t *testing.T) {
	account := validAccount()
	account.ID = ""

	err := ValidateAccount(account)
	if !errors.Is(err, ErrMissingAccountID) {
		t.Fatalf("expected ErrMissingAccountID, got %v", err)
	}
}

func TestValidateAccountRejectsInvalidAllocation(t *testing.T) {
	account := validAccount()
	account.AllocationBps = 10001

	err := ValidateAccount(account)
	if !errors.Is(err, ErrInvalidAllocationBps) {
		t.Fatalf("expected ErrInvalidAllocationBps, got %v", err)
	}
}

func TestValidateAccountRejectsNegativeCashBalance(t *testing.T) {
	account := validAccount()
	account.CashBalanceCents = -1

	err := ValidateAccount(account)
	if !errors.Is(err, ErrInvalidCashBalance) {
		t.Fatalf("expected ErrInvalidCashBalance, got %v", err)
	}
}

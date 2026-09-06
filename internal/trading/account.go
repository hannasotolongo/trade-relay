package trading

type AccountStatus string

const (
	AccountActive   AccountStatus = "ACTIVE"
	AccountDisabled AccountStatus = "DISABLED"
)

type Account struct {
	ID               string
	BrokerAccountID  string
	Status           AccountStatus
	CashBalanceCents int64
	AllocationBps    int64
	MaxPositionCents int64
}

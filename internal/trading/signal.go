package trading

import "time"

type Side string

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

type Signal struct {
	ID         string
	StrategyID string
	Symbol     string
	Side       Side
	Quantity   int64
	CreatedAt  time.Time
}

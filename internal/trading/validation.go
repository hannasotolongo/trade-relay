package trading

import (
	"errors"
	"strings"
)

var (
	ErrMissingSignalID   = errors.New("signal ID is required")
	ErrMissingStrategyID = errors.New("strategy ID is required")
	ErrMissingSymbol     = errors.New("symbol is required")
	ErrInvalidSide       = errors.New("side must be BUY or SELL")
	ErrInvalidQuantity   = errors.New("quantity must be greater than zero")
)

func ValidateSignal(signal Signal) error {
	if strings.TrimSpace(signal.ID) == "" {
		return ErrMissingSignalID
	}

	if strings.TrimSpace(signal.StrategyID) == "" {
		return ErrMissingStrategyID
	}

	if strings.TrimSpace(signal.Symbol) == "" {
		return ErrMissingSymbol
	}

	if signal.Side != SideBuy && signal.Side != SideSell {
		return ErrInvalidSide
	}

	if signal.Quantity <= 0 {
		return ErrInvalidQuantity
	}

	return nil
}

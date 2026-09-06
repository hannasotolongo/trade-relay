package trading

import (
	"errors"
	"testing"
	"time"
)

func validSignal() Signal {
	return Signal{
		ID:         "sig-001",
		StrategyID: "strategy-001",
		Symbol:     "AAPL",
		Side:       SideBuy,
		Quantity:   100,
		CreatedAt:  time.Now(),
	}
}

func TestValidateSignalAcceptsValidSignal(t *testing.T) {
	signal := validSignal()

	err := ValidateSignal(signal)
	if err != nil {
		t.Fatalf("expected valid signal, got error: %v", err)
	}
}

func TestValidateSignalRejectsMissingID(t *testing.T) {
	signal := validSignal()
	signal.ID = ""

	err := ValidateSignal(signal)
	if !errors.Is(err, ErrMissingSignalID) {
		t.Fatalf("expected ErrMissingSignalID, got: %v", err)
	}
}

func TestValidateSignalRejectsInvalidSide(t *testing.T) {
	signal := validSignal()
	signal.Side = Side("HOLD")

	err := ValidateSignal(signal)
	if !errors.Is(err, ErrInvalidSide) {
		t.Fatalf("expected ErrInvalidSide, got: %v", err)
	}
}

func TestValidateSignalRejectsInvalidQuantity(t *testing.T) {
	signal := validSignal()
	signal.Quantity = 0

	err := ValidateSignal(signal)
	if !errors.Is(err, ErrInvalidQuantity) {
		t.Fatalf("expected ErrInvalidQuantity, got: %v", err)
	}
}

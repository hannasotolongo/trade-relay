package trading

import (
	"errors"
	"testing"
)

func validOrderForTest() Order {
	return Order{
		ID:              "order-001",
		SignalID:        "signal-001",
		AccountID:       "account-001",
		BrokerAccountID: "broker-account-001",
		Symbol:          "AAPL",
		Side:            SideBuy,
		Quantity:        100,
		FilledQuantity:  0,
		Status:          OrderCreated,
	}
}

func TestValidateOrderAcceptsValidOrder(t *testing.T) {
	order := validOrderForTest()

	if err := ValidateOrder(order); err != nil {
		t.Fatalf("validate order: %v", err)
	}
}

func TestValidateOrderRejectsMissingOrderID(t *testing.T) {
	order := validOrderForTest()
	order.ID = ""

	err := ValidateOrder(order)

	if !errors.Is(err, ErrMissingOrderID) {
		t.Fatalf("expected %v, got %v", ErrMissingOrderID, err)
	}
}

func TestValidateOrderRejectsMissingSignalID(t *testing.T) {
	order := validOrderForTest()
	order.SignalID = ""

	err := ValidateOrder(order)

	if !errors.Is(err, ErrMissingSignalID) {
		t.Fatalf("expected %v, got %v", ErrMissingSignalID, err)
	}
}

func TestValidateOrderRejectsMissingAccountID(t *testing.T) {
	order := validOrderForTest()
	order.AccountID = ""

	err := ValidateOrder(order)

	if !errors.Is(err, ErrMissingOrderAccountID) {
		t.Fatalf("expected %v, got %v", ErrMissingOrderAccountID, err)
	}
}

func TestValidateOrderRejectsMissingBrokerAccountID(t *testing.T) {
	order := validOrderForTest()
	order.BrokerAccountID = ""

	err := ValidateOrder(order)

	if !errors.Is(err, ErrMissingOrderBrokerID) {
		t.Fatalf("expected %v, got %v", ErrMissingOrderBrokerID, err)
	}
}

func TestValidateOrderRejectsMissingSymbol(t *testing.T) {
	order := validOrderForTest()
	order.Symbol = ""

	err := ValidateOrder(order)

	if !errors.Is(err, ErrMissingOrderSymbol) {
		t.Fatalf("expected %v, got %v", ErrMissingOrderSymbol, err)
	}
}

func TestValidateOrderRejectsInvalidSide(t *testing.T) {
	order := validOrderForTest()
	order.Side = Side("HOLD")

	err := ValidateOrder(order)

	if !errors.Is(err, ErrInvalidOrderSide) {
		t.Fatalf("expected %v, got %v", ErrInvalidOrderSide, err)
	}
}

func TestValidateOrderRejectsZeroQuantity(t *testing.T) {
	order := validOrderForTest()
	order.Quantity = 0

	err := ValidateOrder(order)

	if !errors.Is(err, ErrInvalidOrderQuantity) {
		t.Fatalf("expected %v, got %v", ErrInvalidOrderQuantity, err)
	}
}

func TestValidateOrderRejectsNegativeFilledQuantity(t *testing.T) {
	order := validOrderForTest()
	order.FilledQuantity = -1

	err := ValidateOrder(order)

	if !errors.Is(err, ErrInvalidFilledQuantity) {
		t.Fatalf("expected %v, got %v", ErrInvalidFilledQuantity, err)
	}
}

func TestValidateOrderRejectsFilledQuantityGreaterThanQuantity(t *testing.T) {
	order := validOrderForTest()
	order.FilledQuantity = 101

	err := ValidateOrder(order)

	if !errors.Is(err, ErrInvalidFilledQuantity) {
		t.Fatalf("expected %v, got %v", ErrInvalidFilledQuantity, err)
	}
}

func TestValidateOrderRejectsInvalidStatus(t *testing.T) {
	order := validOrderForTest()
	order.Status = OrderStatus("INVALID")

	err := ValidateOrder(order)

	if !errors.Is(err, ErrInvalidOrderStatus) {
		t.Fatalf("expected %v, got %v", ErrInvalidOrderStatus, err)
	}
}

package store

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/hannasotolongo/trade-relay/internal/trading"
)

func testMySQLOrder() trading.Order {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	return trading.Order{
		ID:              "order-001",
		SignalID:        "signal-001",
		AccountID:       "account-001",
		BrokerAccountID: "broker-account-001",
		BrokerOrderID:   "broker-order-001",
		Symbol:          "AAPL",
		Side:            trading.SideBuy,
		Quantity:        100,
		FilledQuantity:  0,
		Status:          trading.OrderAcknowledged,
		Version:         1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func TestMySQLOrderStoreCreate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer db.Close()

	store := NewMySQLOrderStore(db)
	order := testMySQLOrder()

	mock.ExpectExec("INSERT INTO orders").
		WithArgs(
			order.ID,
			order.SignalID,
			order.AccountID,
			order.BrokerAccountID,
			order.BrokerOrderID,
			order.Symbol,
			order.Side,
			order.Quantity,
			order.FilledQuantity,
			order.Status,
			order.Version,
			order.CreatedAt,
			order.UpdatedAt,
		).
		WillReturnResult(
			sqlmock.NewResult(1, 1),
		)

	if err := store.Create(order); err != nil {
		t.Fatalf("create order: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestMySQLOrderStoreCreateDefaultsVersionToOne(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer db.Close()

	store := NewMySQLOrderStore(db)
	order := testMySQLOrder()
	order.Version = 0

	mock.ExpectExec("INSERT INTO orders").
		WithArgs(
			order.ID,
			order.SignalID,
			order.AccountID,
			order.BrokerAccountID,
			order.BrokerOrderID,
			order.Symbol,
			order.Side,
			order.Quantity,
			order.FilledQuantity,
			order.Status,
			int64(1),
			order.CreatedAt,
			order.UpdatedAt,
		).
		WillReturnResult(
			sqlmock.NewResult(1, 1),
		)

	if err := store.Create(order); err != nil {
		t.Fatalf("create order: %v", err)
	}
}

func TestMySQLOrderStoreGet(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer db.Close()

	store := NewMySQLOrderStore(db)
	expected := testMySQLOrder()

	rows := sqlmock.NewRows([]string{
		"id",
		"signal_id",
		"account_id",
		"broker_account_id",
		"broker_order_id",
		"symbol",
		"side",
		"quantity",
		"filled_quantity",
		"status",
		"version",
		"created_at",
		"updated_at",
	}).AddRow(
		expected.ID,
		expected.SignalID,
		expected.AccountID,
		expected.BrokerAccountID,
		expected.BrokerOrderID,
		expected.Symbol,
		expected.Side,
		expected.Quantity,
		expected.FilledQuantity,
		expected.Status,
		expected.Version,
		expected.CreatedAt,
		expected.UpdatedAt,
	)

	mock.ExpectQuery("SELECT").
		WithArgs(expected.ID).
		WillReturnRows(rows)

	order, err := store.Get(expected.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}

	if order.ID != expected.ID {
		t.Fatalf("expected order ID %q, got %q", expected.ID, order.ID)
	}

	if order.BrokerOrderID != expected.BrokerOrderID {
		t.Fatalf(
			"expected broker order ID %q, got %q",
			expected.BrokerOrderID,
			order.BrokerOrderID,
		)
	}

	if order.Status != expected.Status {
		t.Fatalf(
			"expected status %s, got %s",
			expected.Status,
			order.Status,
		)
	}

	if order.Version != expected.Version {
		t.Fatalf(
			"expected version %d, got %d",
			expected.Version,
			order.Version,
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestMySQLOrderStoreGetMissingOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer db.Close()

	store := NewMySQLOrderStore(db)

	mock.ExpectQuery("SELECT").
		WithArgs("missing-order").
		WillReturnError(sql.ErrNoRows)

	_, err = store.Get("missing-order")

	if !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf(
			"expected %v, got %v",
			ErrOrderNotFound,
			err,
		)
	}
}

func TestMySQLOrderStoreUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer db.Close()

	store := NewMySQLOrderStore(db)
	order := testMySQLOrder()

	order.Status = trading.OrderFilled
	order.FilledQuantity = 100

	mock.ExpectExec("UPDATE orders").
		WithArgs(
			order.BrokerOrderID,
			order.FilledQuantity,
			order.Status,
			order.UpdatedAt,
			order.ID,
			order.Version,
		).
		WillReturnResult(
			sqlmock.NewResult(0, 1),
		)

	if err := store.Update(order); err != nil {
		t.Fatalf("update order: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestMySQLOrderStoreRejectsVersionConflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer db.Close()

	store := NewMySQLOrderStore(db)
	order := testMySQLOrder()

	mock.ExpectExec("UPDATE orders").
		WithArgs(
			order.BrokerOrderID,
			order.FilledQuantity,
			order.Status,
			order.UpdatedAt,
			order.ID,
			order.Version,
		).
		WillReturnResult(
			sqlmock.NewResult(0, 0),
		)

	err = store.Update(order)

	if !errors.Is(err, ErrOrderVersionConflict) {
		t.Fatalf(
			"expected %v, got %v",
			ErrOrderVersionConflict,
			err,
		)
	}
}

func TestMySQLOrderStoreRejectsMissingVersion(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer db.Close()

	store := NewMySQLOrderStore(db)
	order := testMySQLOrder()
	order.Version = 0

	err = store.Update(order)

	if !errors.Is(err, ErrOrderVersionConflict) {
		t.Fatalf(
			"expected %v, got %v",
			ErrOrderVersionConflict,
			err,
		)
	}
}

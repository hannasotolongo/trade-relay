package store

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/hannasotolongo/trade-relay/internal/trading"
)

func TestMySQLOrderStoreIntegration(t *testing.T) {
	dsn := os.Getenv("TRADERELAY_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TRADERELAY_MYSQL_DSN not set")
	}

	ctx := context.Background()

	db, err := OpenMySQL(ctx, dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	defer db.Close()

	store := NewMySQLOrderStore(db)

	orderID := "integration-order-001"

	_, err = db.Exec(
		"DELETE FROM orders WHERE id = ?",
		orderID,
	)
	if err != nil {
		t.Fatalf("delete existing integration order: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.Exec(
			"DELETE FROM orders WHERE id = ?",
			orderID,
		)
	})

	now := time.Now().UTC()

	order := trading.Order{
		ID:              orderID,
		SignalID:        "integration-signal-001",
		AccountID:       "integration-account-001",
		BrokerAccountID: "integration-broker-account-001",
		Symbol:          "AAPL",
		Side:            trading.SideBuy,
		Quantity:        100,
		FilledQuantity:  0,
		Status:          trading.OrderCreated,
		Version:         1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := store.Create(order); err != nil {
		t.Fatalf("create order: %v", err)
	}

	stored, err := store.Get(order.ID)
	if err != nil {
		t.Fatalf("get created order: %v", err)
	}

	if stored.Status != trading.OrderCreated {
		t.Fatalf(
			"expected status %s, got %s",
			trading.OrderCreated,
			stored.Status,
		)
	}

	if stored.Version != 1 {
		t.Fatalf(
			"expected version 1, got %d",
			stored.Version,
		)
	}

	// Keep a stale copy at version 1. We will use it later to prove
	// that optimistic locking rejects an outdated writer.
	stale := stored

	stored.Status = trading.OrderSubmitting
	stored.UpdatedAt = time.Now().UTC()

	if err := store.Update(&stored); err != nil {
		t.Fatalf("update order: %v", err)
	}

	// The Go object should advance immediately after the successful
	// database update.
	if stored.Version != 2 {
		t.Fatalf(
			"expected in-memory version 2 after update, got %d",
			stored.Version,
		)
	}

	updated, err := store.Get(order.ID)
	if err != nil {
		t.Fatalf("get updated order: %v", err)
	}

	if updated.Status != trading.OrderSubmitting {
		t.Fatalf(
			"expected status %s, got %s",
			trading.OrderSubmitting,
			updated.Status,
		)
	}

	if updated.Version != 2 {
		t.Fatalf(
			"expected database version 2, got %d",
			updated.Version,
		)
	}

	// The stale copy still believes the order is version 1.
	// MySQL is now version 2, so this update must be rejected.
	stale.Status = trading.OrderRejected
	stale.UpdatedAt = time.Now().UTC()

	err = store.Update(&stale)

	if !errors.Is(err, ErrOrderVersionConflict) {
		t.Fatalf(
			"expected %v, got %v",
			ErrOrderVersionConflict,
			err,
		)
	}

	// A failed optimistic-lock update must not advance the stale
	// Go object's version.
	if stale.Version != 1 {
		t.Fatalf(
			"expected stale version to remain 1, got %d",
			stale.Version,
		)
	}

	final, err := store.Get(order.ID)
	if err != nil {
		t.Fatalf("get final order: %v", err)
	}

	if final.Status != trading.OrderSubmitting {
		t.Fatalf(
			"expected final status %s, got %s",
			trading.OrderSubmitting,
			final.Status,
		)
	}

	if final.Version != 2 {
		t.Fatalf(
			"expected final version 2, got %d",
			final.Version,
		)
	}
}

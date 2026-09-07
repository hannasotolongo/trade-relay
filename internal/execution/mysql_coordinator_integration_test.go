package execution

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/hannasotolongo/trade-relay/internal/broker"
	"github.com/hannasotolongo/trade-relay/internal/store"
	"github.com/hannasotolongo/trade-relay/internal/trading"
)

func TestCoordinatorWithMySQLIntegration(t *testing.T) {
	dsn := os.Getenv("TRADERELAY_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TRADERELAY_MYSQL_DSN not set")
	}

	ctx := context.Background()

	db, err := store.OpenMySQL(ctx, dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	defer db.Close()

	orderStore := store.NewMySQLOrderStore(db)

	orderID := "integration-coordinator-order-001"

	_, err = db.Exec(
		"DELETE FROM orders WHERE id = ?",
		orderID,
	)
	if err != nil {
		t.Fatalf("delete existing order: %v", err)
	}

	now := time.Date(
		2026,
		time.September,
		6,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	order := trading.Order{
		ID:              orderID,
		SignalID:        "signal-001",
		AccountID:       "acct-001",
		BrokerAccountID: "broker-account-001",
		Symbol:          "AAPL",
		Side:            trading.SideBuy,
		Quantity:        100,
		FilledQuantity:  0,
		Status:          trading.OrderCreated,
		Version:         1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := orderStore.Create(order); err != nil {
		t.Fatalf("create order: %v", err)
	}

	coordinator := Coordinator{
		Broker: &broker.SimulatedBroker{},
		Store:  orderStore,
	}

	if err := coordinator.Submit(
		ctx,
		&order,
		now.Add(time.Second),
	); err != nil {
		t.Fatalf("submit order: %v", err)
	}

	stored, err := orderStore.Get(orderID)
	if err != nil {
		t.Fatalf("get final order: %v", err)
	}

	if stored.Status != trading.OrderAcknowledged {
		t.Fatalf(
			"expected status %s, got %s",
			trading.OrderAcknowledged,
			stored.Status,
		)
	}

	if stored.Version != 3 {
		t.Fatalf(
			"expected version 3, got %d",
			stored.Version,
		)
	}

	if stored.BrokerOrderID == "" {
		t.Fatal("expected broker order ID to be persisted")
	}

	if order.Version != 3 {
		t.Fatalf(
			"expected in-memory order version 3, got %d",
			order.Version,
		)
	}

	if order.Status != trading.OrderAcknowledged {
		t.Fatalf(
			"expected in-memory status %s, got %s",
			trading.OrderAcknowledged,
			order.Status,
		)
	}
}

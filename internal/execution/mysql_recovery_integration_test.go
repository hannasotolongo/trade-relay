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

func TestMySQLRecoveryReconcilesInterruptedSubmission(t *testing.T) {
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

	orderID := "integration-recovery-order-001"

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
		SignalID:        "signal-recovery-001",
		AccountID:       "acct-001",
		BrokerAccountID: "broker-account-001",
		Symbol:          "AAPL",
		Side:            trading.SideBuy,
		Quantity:        100,
		FilledQuantity:  0,
		Status:          trading.OrderSubmitting,
		Version:         2,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := orderStore.Create(order); err != nil {
		t.Fatalf("create interrupted order: %v", err)
	}

	storedBeforeRecovery, err := orderStore.Get(orderID)
	if err != nil {
		t.Fatalf("get interrupted order: %v", err)
	}

	if storedBeforeRecovery.Status != trading.OrderSubmitting {
		t.Fatalf(
			"expected pre-recovery status %s, got %s",
			trading.OrderSubmitting,
			storedBeforeRecovery.Status,
		)
	}

	if storedBeforeRecovery.Version != 2 {
		t.Fatalf(
			"expected pre-recovery version 2, got %d",
			storedBeforeRecovery.Version,
		)
	}

	if storedBeforeRecovery.BrokerOrderID != "" {
		t.Fatalf(
			"expected no broker order ID before recovery, got %s",
			storedBeforeRecovery.BrokerOrderID,
		)
	}

	simulatedBroker := &broker.SimulatedBroker{}

	brokerResult, err := simulatedBroker.SubmitOrder(
		ctx,
		storedBeforeRecovery,
	)
	if err != nil {
		t.Fatalf("submit order to simulated broker: %v", err)
	}

	if brokerResult.OrderID == "" {
		t.Fatal("expected broker to assign an order ID")
	}

	if brokerResult.Status != trading.OrderAcknowledged {
		t.Fatalf(
			"expected broker status %s, got %s",
			trading.OrderAcknowledged,
			brokerResult.Status,
		)
	}

	restartedOrderStore := store.NewMySQLOrderStore(db)

	recoveredOrder, err := restartedOrderStore.Get(orderID)
	if err != nil {
		t.Fatalf("load interrupted order after restart: %v", err)
	}

	reconciler := Reconciler{
		Broker: simulatedBroker,
		Store:  restartedOrderStore,
	}

	if err := reconciler.Reconcile(
		ctx,
		&recoveredOrder,
		now.Add(2*time.Second),
	); err != nil {
		t.Fatalf("reconcile interrupted order: %v", err)
	}

	storedAfterRecovery, err := restartedOrderStore.Get(orderID)
	if err != nil {
		t.Fatalf("get recovered order: %v", err)
	}

	if storedAfterRecovery.Status != trading.OrderAcknowledged {
		t.Fatalf(
			"expected recovered status %s, got %s",
			trading.OrderAcknowledged,
			storedAfterRecovery.Status,
		)
	}

	if storedAfterRecovery.Version != 3 {
		t.Fatalf(
			"expected recovered version 3, got %d",
			storedAfterRecovery.Version,
		)
	}

	if storedAfterRecovery.BrokerOrderID != brokerResult.OrderID {
		t.Fatalf(
			"expected broker order ID %s, got %s",
			brokerResult.OrderID,
			storedAfterRecovery.BrokerOrderID,
		)
	}
}

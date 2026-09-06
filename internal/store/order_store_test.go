package store

import (
	"errors"
	"sync"
	"testing"

	"github.com/hannasotolongo/trade-relay/internal/trading"
)

func testOrder(id string) trading.Order {
	return trading.Order{
		ID:     id,
		Status: trading.OrderCreated,
	}
}

func TestOrderStoreCreateAndGet(t *testing.T) {
	store := NewOrderStore()
	order := testOrder("order-001")

	if err := store.Create(order); err != nil {
		t.Fatalf("create order: %v", err)
	}

	got, err := store.Get(order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}

	if got.ID != order.ID {
		t.Fatalf("expected order ID %q, got %q", order.ID, got.ID)
	}
}

func TestOrderStoreRejectsDuplicate(t *testing.T) {
	store := NewOrderStore()
	order := testOrder("order-001")

	if err := store.Create(order); err != nil {
		t.Fatalf("create order: %v", err)
	}

	err := store.Create(order)
	if !errors.Is(err, ErrOrderAlreadyExists) {
		t.Fatalf("expected ErrOrderAlreadyExists, got %v", err)
	}
}

func TestOrderStoreReturnsNotFound(t *testing.T) {
	store := NewOrderStore()

	_, err := store.Get("missing-order")
	if !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("expected ErrOrderNotFound, got %v", err)
	}
}

func TestOrderStoreUpdatesOrder(t *testing.T) {
	store := NewOrderStore()
	order := testOrder("order-001")

	if err := store.Create(order); err != nil {
		t.Fatalf("create order: %v", err)
	}

	order.Status = trading.OrderAcknowledged

	if err := store.Update(order); err != nil {
		t.Fatalf("update order: %v", err)
	}

	got, err := store.Get(order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}

	if got.Status != trading.OrderAcknowledged {
		t.Fatalf(
			"expected status %s, got %s",
			trading.OrderAcknowledged,
			got.Status,
		)
	}
}

func TestOrderStoreConcurrentDuplicateCreate(t *testing.T) {
	store := NewOrderStore()
	order := testOrder("order-001")

	const attempts = 20

	var wg sync.WaitGroup
	results := make(chan error, attempts)

	for i := 0; i < attempts; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			results <- store.Create(order)
		}()
	}

	wg.Wait()
	close(results)

	successes := 0
	duplicates := 0

	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrOrderAlreadyExists):
			duplicates++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if successes != 1 {
		t.Fatalf("expected 1 successful create, got %d", successes)
	}

	if duplicates != attempts-1 {
		t.Fatalf(
			"expected %d duplicate errors, got %d",
			attempts-1,
			duplicates,
		)
	}
}

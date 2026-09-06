package store

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hannasotolongo/trade-relay/internal/trading"
)

func testSignal() trading.Signal {
	return trading.Signal{
		ID:         "sig-001",
		StrategyID: "strategy-001",
		Symbol:     "AAPL",
		Side:       trading.SideBuy,
		Quantity:   100,
		CreatedAt:  time.Now(),
	}
}

func TestSignalStoreCreateAndGet(t *testing.T) {
	store := NewSignalStore()
	signal := testSignal()

	if err := store.Create(signal); err != nil {
		t.Fatalf("create signal: %v", err)
	}

	got, exists := store.Get(signal.ID)
	if !exists {
		t.Fatal("expected signal to exist")
	}

	if got.ID != signal.ID {
		t.Fatalf("expected signal ID %q, got %q", signal.ID, got.ID)
	}
}

func TestSignalStoreRejectsDuplicateSignal(t *testing.T) {
	store := NewSignalStore()
	signal := testSignal()

	if err := store.Create(signal); err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	err := store.Create(signal)
	if !errors.Is(err, ErrSignalAlreadyExists) {
		t.Fatalf("expected ErrSignalAlreadyExists, got %v", err)
	}
}

func TestSignalStoreConcurrentDuplicateCreate(t *testing.T) {
	store := NewSignalStore()
	signal := testSignal()

	const attempts = 20

	var wg sync.WaitGroup
	wg.Add(attempts)

	results := make(chan error, attempts)

	for i := 0; i < attempts; i++ {
		go func() {
			defer wg.Done()
			results <- store.Create(signal)
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
		case errors.Is(err, ErrSignalAlreadyExists):
			duplicates++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if successes != 1 {
		t.Fatalf("expected exactly 1 successful create, got %d", successes)
	}

	if duplicates != attempts-1 {
		t.Fatalf("expected %d duplicate errors, got %d", attempts-1, duplicates)
	}
}

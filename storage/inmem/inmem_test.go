package inmem

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestInMemoryStore_SetAndGet(t *testing.T) {
	store := NewInMemoryStore()
	defer store.Close()
	ctx := context.Background()

	err := store.Set(ctx, "key1", 42.0, time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := store.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != 42.0 {
		t.Fatalf("expected 42.0, got %f", val)
	}
}

func TestInMemoryStore_GetMissing(t *testing.T) {
	store := NewInMemoryStore()
	defer store.Close()
	ctx := context.Background()

	val, err := store.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != 0 {
		t.Fatalf("expected 0 for missing key, got %f", val)
	}
}

func TestInMemoryStore_GetExpired(t *testing.T) {
	store := NewInMemoryStore()
	defer store.Close()
	ctx := context.Background()

	err := store.Set(ctx, "expiring", 10.0, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	val, err := store.Get(ctx, "expiring")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != 0 {
		t.Fatalf("expected 0 for expired key, got %f", val)
	}
}

func TestInMemoryStore_Incr_NewKey(t *testing.T) {
	store := NewInMemoryStore()
	defer store.Close()
	ctx := context.Background()

	val, err := store.Incr(ctx, "counter", time.Minute)
	if err != nil {
		t.Fatalf("Incr failed: %v", err)
	}
	if val != 1 {
		t.Fatalf("expected 1, got %f", val)
	}
}

func TestInMemoryStore_Incr_ExistingKey(t *testing.T) {
	store := NewInMemoryStore()
	defer store.Close()
	ctx := context.Background()

	store.Incr(ctx, "counter", time.Minute)
	store.Incr(ctx, "counter", time.Minute)
	val, err := store.Incr(ctx, "counter", time.Minute)
	if err != nil {
		t.Fatalf("Incr failed: %v", err)
	}
	if val != 3 {
		t.Fatalf("expected 3, got %f", val)
	}
}

func TestInMemoryStore_Incr_ExpiredKey(t *testing.T) {
	store := NewInMemoryStore()
	defer store.Close()
	ctx := context.Background()

	store.Incr(ctx, "counter", 50*time.Millisecond)
	store.Incr(ctx, "counter", 50*time.Millisecond)

	time.Sleep(100 * time.Millisecond)

	val, err := store.Incr(ctx, "counter", time.Minute)
	if err != nil {
		t.Fatalf("Incr failed: %v", err)
	}
	if val != 1 {
		t.Fatalf("expected 1 after expiry, got %f", val)
	}
}

func TestInMemoryStore_SetOverwrite(t *testing.T) {
	store := NewInMemoryStore()
	defer store.Close()
	ctx := context.Background()

	store.Set(ctx, "key", 10.0, time.Minute)
	store.Set(ctx, "key", 20.0, time.Minute)

	val, err := store.Get(ctx, "key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != 20.0 {
		t.Fatalf("expected 20.0, got %f", val)
	}
}

func TestInMemoryStore_Concurrency(t *testing.T) {
	store := NewInMemoryStore()
	defer store.Close()
	ctx := context.Background()

	var wg sync.WaitGroup
	n := 100
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.Incr(ctx, "concurrent", time.Minute)
		}()
	}
	wg.Wait()

	val, _ := store.Get(ctx, "concurrent")
	if val != float64(n) {
		t.Fatalf("expected %d, got %f", n, val)
	}
}

func TestInMemoryStore_GC(t *testing.T) {
	s := &inMemoryStore{
		done: make(chan struct{}),
	}
	ctx := context.Background()

	// Don't start default GC, we'll call removeExpired manually
	s.Set(ctx, "alive", 1.0, time.Hour)
	s.Set(ctx, "dead", 2.0, time.Millisecond)

	time.Sleep(50 * time.Millisecond)
	s.removeExpired()

	val, _ := s.Get(ctx, "alive")
	if val != 1.0 {
		t.Fatalf("alive key should still exist, got %f", val)
	}
	val, _ = s.Get(ctx, "dead")
	if val != 0 {
		t.Fatalf("dead key should be removed, got %f", val)
	}

	close(s.done)
}

func TestInMemoryStore_Close(t *testing.T) {
	store := NewInMemoryStore()
	err := store.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

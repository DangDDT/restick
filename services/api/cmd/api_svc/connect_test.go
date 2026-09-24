package main

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eykh/restick/api_svc/internal/infrastructure/postgres"
)

func TestConnectWithRetry_SuccessFirstAttempt(t *testing.T) {
	dummyStore := &postgres.Store{}
	var attempts int32

	connector := func(ctx context.Context, dsn string) (*postgres.Store, error) {
		atomic.AddInt32(&attempts, 1)
		return dummyStore, nil
	}

	store, err := connectWithRetry(context.Background(), connector, "postgres://user:pass@localhost:5432/db", 12, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if store != dummyStore {
		t.Fatalf("expected returned store pointer to match")
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Fatalf("expected exactly 1 attempt, got %d", got)
	}
}

func TestConnectWithRetry_SuccessAfterTransientFailures(t *testing.T) {
	dummyStore := &postgres.Store{}
	var attempts int32
	transientErr := errors.New("transient dns/connection failure")

	connector := func(ctx context.Context, dsn string) (*postgres.Store, error) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 4 {
			return nil, transientErr
		}
		return dummyStore, nil
	}

	store, err := connectWithRetry(context.Background(), connector, "postgres://user:pass@localhost:5432/db", 12, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if store != dummyStore {
		t.Fatalf("expected returned store pointer to match")
	}
	if got := atomic.LoadInt32(&attempts); got != 4 {
		t.Fatalf("expected exactly 4 attempts before success, got %d", got)
	}
}

func TestConnectWithRetry_ExhaustionReturnsWrappedError(t *testing.T) {
	var attempts int32
	fatalErr := errors.New("connection refused")

	connector := func(ctx context.Context, dsn string) (*postgres.Store, error) {
		atomic.AddInt32(&attempts, 1)
		return nil, fatalErr
	}

	const maxAttempts = 5
	store, err := connectWithRetry(context.Background(), connector, "postgres://user:pass@localhost:5432/db", maxAttempts, 5*time.Millisecond)
	if store != nil {
		t.Fatalf("expected nil store on failure, got %v", store)
	}
	if err == nil {
		t.Fatal("expected non-nil error after exhaustion")
	}
	if !errors.Is(err, fatalErr) {
		t.Fatalf("expected wrapped error to match target error using errors.Is; got %v", err)
	}
	if !strings.Contains(err.Error(), "5") {
		t.Fatalf("expected error message to mention attempt count, got %q", err.Error())
	}
	if got := atomic.LoadInt32(&attempts); got != maxAttempts {
		t.Fatalf("expected exactly %d attempts, got %d", maxAttempts, got)
	}
}

func TestConnectWithRetry_CancellationDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var attempts int32
	transientErr := errors.New("database not ready")

	connector := func(c context.Context, dsn string) (*postgres.Store, error) {
		atomic.AddInt32(&attempts, 1)
		// Trigger cancel right after first failed attempt
		cancel()
		return nil, transientErr
	}

	store, err := connectWithRetry(ctx, connector, "postgres://user:pass@localhost:5432/db", 12, 500*time.Millisecond)
	if store != nil {
		t.Fatalf("expected nil store on cancellation, got %v", store)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled error, got %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Fatalf("expected exactly 1 attempt before cancellation aborted backoff, got %d", got)
	}
}

func TestConnectWithRetry_ContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var attempts int32
	connector := func(c context.Context, dsn string) (*postgres.Store, error) {
		atomic.AddInt32(&attempts, 1)
		return &postgres.Store{}, nil
	}

	store, err := connectWithRetry(ctx, connector, "postgres://user:pass@localhost:5432/db", 12, 10*time.Millisecond)
	if store != nil {
		t.Fatalf("expected nil store on pre-cancelled context, got %v", store)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled error, got %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 0 {
		t.Fatalf("expected 0 attempts for pre-cancelled context, got %d", got)
	}
}

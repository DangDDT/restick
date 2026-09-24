package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/eykh/restick/api_svc/internal/infrastructure/postgres"
)

const (
	defaultPostgresConnectAttempts = 12
	defaultPostgresConnectDelay    = 1 * time.Second
)

type PostgresConnector func(ctx context.Context, dsn string) (*postgres.Store, error)

func connectWithRetry(
	ctx context.Context,
	connector PostgresConnector,
	dsn string,
	maxAttempts int,
	delay time.Duration,
) (*postgres.Store, error) {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		store, err := connector(ctx, dsn)
		if err == nil {
			return store, nil
		}

		lastErr = err
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if attempt == maxAttempts {
			break
		}

		slog.Warn("postgres connect failed, retrying",
			"attempt", attempt,
			"max_attempts", maxAttempts,
			"retry_in", delay,
			"err", err,
		)

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}

	return nil, fmt.Errorf("postgres connect failed after %d attempts: %w", maxAttempts, lastErr)
}

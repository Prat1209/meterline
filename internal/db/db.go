// Package db opens the Postgres pool and applies migrations.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/Prat1209/meterline/migrations"
)

const (
	connectTimeout = 5 * time.Second
	pingAttempts   = 5
	initialBackoff = 500 * time.Millisecond
)

// Connect opens a pool and waits until Postgres answers a ping, retrying with
// exponential backoff so the service survives starting before the database.
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	cfg.ConnConfig.ConnectTimeout = connectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	backoff := initialBackoff
	for attempt := 1; ; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, connectTimeout)
		err = pool.Ping(pingCtx)
		cancel()
		if err == nil {
			return pool, nil
		}
		if attempt == pingAttempts {
			pool.Close()
			return nil, fmt.Errorf("ping postgres after %d attempts: %w", attempt, err)
		}
		select {
		case <-time.After(backoff):
			backoff *= 2
		case <-ctx.Done():
			pool.Close()
			return nil, ctx.Err()
		}
	}
}

// Migrate applies all pending migrations embedded in the binary.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer func() { _ = sqlDB.Close() }() // closes the sql.DB wrapper only, not the pool

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

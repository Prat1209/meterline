// Command seed creates a demo account and prints its API key once.
//
//	go run ./cmd/seed "Acme AI"
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Prat1209/meterline/internal/auth"
	"github.com/Prat1209/meterline/internal/config"
	"github.com/Prat1209/meterline/internal/db"
)

func main() {
	name := "Demo account"
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	if err := run(name); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
}

func run(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		return err
	}

	key, hash, err := auth.NewKey()
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}
	var id int64
	err = pool.QueryRow(ctx,
		`INSERT INTO accounts (name, api_key_hash) VALUES ($1, $2) RETURNING id`, name, hash).Scan(&id)
	if err != nil {
		return fmt.Errorf("create account: %w", err)
	}

	fmt.Printf("account %d (%s)\nAPI key (shown once, not stored): %s\n", id, name, key)
	return nil
}

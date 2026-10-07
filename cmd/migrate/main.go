// Command migrate applies pending database migrations and exits.
// The API also migrates on startup; this is for running it on its own.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Prat1209/meterline/internal/config"
	"github.com/Prat1209/meterline/internal/db"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
	fmt.Println("migrations applied")
}

func run() error {
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
	return db.Migrate(ctx, pool)
}

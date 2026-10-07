// Package config loads service settings from environment variables.
// Secrets are never read from files or flags, only from the environment.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port            int
	DatabaseURL     string
	StripeSecretKey string // optional until M2
}

// Load reads the environment and fails fast on anything missing or unsafe.
func Load() (Config, error) {
	return load(os.Getenv)
}

func load(getenv func(string) string) (Config, error) {
	var errs []error

	port := 8080
	if v := getenv("PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 || p > 65535 {
			errs = append(errs, fmt.Errorf("PORT must be a number between 1 and 65535, got %q", v))
		} else {
			port = p
		}
	}

	dbURL := getenv("DATABASE_URL")
	if dbURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}

	// Test mode only: refuse to start with a live key.
	stripeKey := getenv("STRIPE_SECRET_KEY")
	if stripeKey != "" && !strings.HasPrefix(stripeKey, "sk_test_") {
		errs = append(errs, errors.New("STRIPE_SECRET_KEY must be a test-mode key (sk_test_...)"))
	}

	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}
	return Config{Port: port, DatabaseURL: dbURL, StripeSecretKey: stripeKey}, nil
}

// Package auth implements API-key authentication for the demo.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"math/big"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Prat1209/meterline/internal/httperr"
)

const keyPrefix = "mk_test_"

type ctxKey struct{}

// NewKey returns a fresh API key and the hash to store for it.
func NewKey() (key string, hash []byte, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, err
	}
	key = keyPrefix + new(big.Int).SetBytes(b).Text(62)
	return key, HashKey(key), nil
}

// HashKey is the value stored in accounts.api_key_hash.
func HashKey(key string) []byte {
	sum := sha256.Sum256([]byte(key))
	return sum[:]
}

// AccountID returns the authenticated account set by Middleware.
func AccountID(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(ctxKey{}).(int64)
	return id, ok
}

// Middleware rejects requests without a valid "Authorization: Bearer <key>" header.
func Middleware(pool *pgxpool.Pool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || !strings.HasPrefix(key, keyPrefix) {
			httperr.Write(w, http.StatusUnauthorized, "unauthorized", "missing or malformed API key")
			return
		}

		var id int64
		err := pool.QueryRow(r.Context(),
			`SELECT id FROM accounts WHERE api_key_hash = $1`, HashKey(key)).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			httperr.Write(w, http.StatusUnauthorized, "unauthorized", "invalid API key")
			return
		}
		if err != nil {
			httperr.Write(w, http.StatusInternalServerError, "internal_error", "could not verify API key")
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, id)))
	})
}

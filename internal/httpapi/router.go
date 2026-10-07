// Package httpapi wires HTTP routes to their handlers.
package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Prat1209/meterline/internal/auth"
	"github.com/Prat1209/meterline/internal/usage"
)

// NewRouter returns the service's HTTP handler.
func NewRouter(pool *pgxpool.Pool, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", Healthz)
	mux.Handle("POST /v1/usage", auth.Middleware(pool, usage.NewHandler(usage.NewStore(pool), logger)))
	return mux
}

// Healthz reports that the process is up. It does not touch the database.
func Healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

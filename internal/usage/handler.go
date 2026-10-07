// Package usage records metered token usage with idempotency guarantees.
package usage

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Prat1209/meterline/internal/auth"
	"github.com/Prat1209/meterline/internal/httperr"
)

const dbTimeout = 5 * time.Second

type Handler struct {
	store  *Store
	logger *slog.Logger
	now    func() time.Time
}

func NewHandler(store *Store, logger *slog.Logger) *Handler {
	return &Handler{store: store, logger: logger, now: time.Now}
}

// ServeHTTP handles POST /v1/usage. It must run behind auth.Middleware.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	accountID, ok := auth.AccountID(r.Context())
	if !ok {
		httperr.Write(w, http.StatusUnauthorized, "unauthorized", "missing account")
		return
	}

	key := r.Header.Get("Idempotency-Key")
	if !ValidIdempotencyKey(key) {
		httperr.Write(w, http.StatusBadRequest, "invalid_request", "Idempotency-Key header is required (1-255 characters)")
		return
	}

	req, err := Decode(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err != nil {
		httperr.Write(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()

	res, err := h.store.Record(ctx, accountID, key, req.Hash(), req.Event(h.now()))
	switch {
	case errors.Is(err, ErrKeyReused):
		httperr.Write(w, http.StatusUnprocessableEntity, "idempotency_key_reused",
			"this Idempotency-Key was already used with a different request body")
		return
	case err != nil:
		h.logger.Error("record usage", "err", err, "account_id", accountID)
		httperr.Write(w, http.StatusInternalServerError, "internal_error", "could not record usage")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if res.Replayed {
		w.Header().Set("Idempotent-Replayed", "true")
	}
	w.WriteHeader(res.Status)
	_, _ = w.Write(res.Body)
}

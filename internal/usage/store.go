package usage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrKeyReused means the idempotency key was already used with a different body.
var ErrKeyReused = errors.New("idempotency key reused with a different request")

// Result is the response to send, either freshly created or replayed.
type Result struct {
	Status   int
	Body     []byte
	Replayed bool
}

// eventResponse is the JSON returned for a recorded usage event.
type eventResponse struct {
	ID           int64     `json:"id"`
	AccountID    int64     `json:"account_id"`
	Model        string    `json:"model"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
	OccurredAt   time.Time `json:"occurred_at"`
	CreatedAt    time.Time `json:"created_at"`
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Record stores a usage event at most once per (account, idempotency key).
//
// Everything happens in one transaction. The idempotency row is inserted
// first; if a concurrent request holds the same key, Postgres blocks that
// INSERT on the primary key until the other transaction finishes, so
// duplicates serialize without an explicit lock or "in progress" state.
func (s *Store) Record(ctx context.Context, accountID int64, key string, reqHash []byte, ev Event) (Result, error) {
	var res Result
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			INSERT INTO idempotency_keys (account_id, key, request_hash, response_status, response_body)
			VALUES ($1, $2, $3, 0, ''::bytea)
			ON CONFLICT (account_id, key) DO NOTHING`,
			accountID, key, reqHash)
		if err != nil {
			return fmt.Errorf("claim idempotency key: %w", err)
		}

		if tag.RowsAffected() == 0 {
			res, err = replay(ctx, tx, accountID, key, reqHash)
			return err
		}

		res, err = create(ctx, tx, accountID, key, ev)
		return err
	})
	return res, err
}

func replay(ctx context.Context, tx pgx.Tx, accountID int64, key string, reqHash []byte) (Result, error) {
	var storedHash, body []byte
	var status int
	err := tx.QueryRow(ctx, `
		SELECT request_hash, response_status, response_body
		FROM idempotency_keys WHERE account_id = $1 AND key = $2`,
		accountID, key).Scan(&storedHash, &status, &body)
	if err != nil {
		return Result{}, fmt.Errorf("load idempotency key: %w", err)
	}
	if !bytes.Equal(storedHash, reqHash) {
		return Result{}, ErrKeyReused
	}
	return Result{Status: status, Body: body, Replayed: true}, nil
}

func create(ctx context.Context, tx pgx.Tx, accountID int64, key string, ev Event) (Result, error) {
	resp := eventResponse{
		AccountID:    accountID,
		Model:        ev.Model,
		InputTokens:  ev.InputTokens,
		OutputTokens: ev.OutputTokens,
	}
	err := tx.QueryRow(ctx, `
		INSERT INTO usage_events (account_id, idempotency_key, model, input_tokens, output_tokens, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, occurred_at, created_at`,
		accountID, key, ev.Model, ev.InputTokens, ev.OutputTokens, ev.OccurredAt,
	).Scan(&resp.ID, &resp.OccurredAt, &resp.CreatedAt)
	if err != nil {
		return Result{}, fmt.Errorf("insert usage event: %w", err)
	}
	resp.OccurredAt = resp.OccurredAt.UTC()
	resp.CreatedAt = resp.CreatedAt.UTC()

	body, err := json.Marshal(resp)
	if err != nil {
		return Result{}, fmt.Errorf("encode response: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE idempotency_keys SET response_status = $3, response_body = $4
		WHERE account_id = $1 AND key = $2`,
		accountID, key, http.StatusCreated, body)
	if err != nil {
		return Result{}, fmt.Errorf("save idempotent response: %w", err)
	}
	return Result{Status: http.StatusCreated, Body: body}, nil
}

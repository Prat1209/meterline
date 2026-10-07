package usage_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Prat1209/meterline/internal/auth"
	"github.com/Prat1209/meterline/internal/db"
	"github.com/Prat1209/meterline/internal/httpapi"
)

// testEnv runs the real router against a real Postgres from DATABASE_URL.
type testEnv struct {
	pool   *pgxpool.Pool
	server *httptest.Server
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("DATABASE_URL must be set in CI")
		}
		t.Skip("DATABASE_URL not set; start Postgres with `docker compose up -d postgres` to run integration tests")
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(httpapi.NewRouter(pool, slog.New(slog.NewTextHandler(io.Discard, nil))))
	t.Cleanup(srv.Close)
	return &testEnv{pool: pool, server: srv}
}

// newAccount creates an isolated account so tests never share rows.
func (e *testEnv) newAccount(t *testing.T) (id int64, key string) {
	t.Helper()
	key, hash, err := auth.NewKey()
	if err != nil {
		t.Fatal(err)
	}
	err = e.pool.QueryRow(context.Background(),
		`INSERT INTO accounts (name, api_key_hash) VALUES ($1, $2) RETURNING id`, t.Name(), hash).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id, key
}

func (e *testEnv) post(t *testing.T, apiKey, idemKey, body string) (*http.Response, []byte) {
	t.Helper()
	resp, b, err := e.tryPost(apiKey, idemKey, body)
	if err != nil {
		t.Fatal(err)
	}
	return resp, b
}

// tryPost is safe to call from goroutines (it never calls t.Fatal).
func (e *testEnv) tryPost(apiKey, idemKey, body string) (*http.Response, []byte, error) {
	req, err := http.NewRequest(http.MethodPost, e.server.URL+"/v1/usage", bytes.NewBufferString(body))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	return resp, b, err
}

func (e *testEnv) countEvents(t *testing.T, accountID int64) int {
	t.Helper()
	var n int
	err := e.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM usage_events WHERE account_id = $1`, accountID).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

const body = `{"model":"gpt-x","input_tokens":1200,"output_tokens":340,"occurred_at":"2026-10-07T17:00:00Z"}`

func TestReplayReturnsSameResponseAndOneRow(t *testing.T) {
	e := newTestEnv(t)
	acct, key := e.newAccount(t)

	first, firstBody := e.post(t, key, "req-1", body)
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("first status = %d, body %s", first.StatusCode, firstBody)
	}
	if first.Header.Get("Idempotent-Replayed") != "" {
		t.Error("first response should not be marked as a replay")
	}

	second, secondBody := e.post(t, key, "req-1", body)
	if second.StatusCode != http.StatusCreated {
		t.Fatalf("replay status = %d, body %s", second.StatusCode, secondBody)
	}
	if second.Header.Get("Idempotent-Replayed") != "true" {
		t.Error("replay should set Idempotent-Replayed: true")
	}
	if !bytes.Equal(firstBody, secondBody) {
		t.Errorf("replay body differs:\nfirst:  %s\nreplay: %s", firstBody, secondBody)
	}
	if n := e.countEvents(t, acct); n != 1 {
		t.Errorf("usage rows = %d, want 1", n)
	}
}

func TestSameKeyDifferentBodyIsRejected(t *testing.T) {
	e := newTestEnv(t)
	acct, key := e.newAccount(t)

	if resp, b := e.post(t, key, "req-1", body); resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, body %s", resp.StatusCode, b)
	}
	other := `{"model":"gpt-x","input_tokens":9999,"output_tokens":340,"occurred_at":"2026-10-07T17:00:00Z"}`
	resp, b := e.post(t, key, "req-1", other)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422, body %s", resp.StatusCode, b)
	}
	if n := e.countEvents(t, acct); n != 1 {
		t.Errorf("usage rows = %d, want 1", n)
	}
}

func TestConcurrentDuplicatesCreateOneRow(t *testing.T) {
	e := newTestEnv(t)
	acct, key := e.newAccount(t)

	const n = 100
	var wg sync.WaitGroup
	statuses := make([]int, n)
	bodies := make([][]byte, n)
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, b, err := e.tryPost(key, "same-key", body)
			if err != nil {
				errs[i] = err
				return
			}
			statuses[i], bodies[i] = resp.StatusCode, b
		}()
	}
	wg.Wait()

	for i := range n {
		if errs[i] != nil {
			t.Fatalf("request %d: %v", i, errs[i])
		}
		if statuses[i] != http.StatusCreated {
			t.Fatalf("request %d: status %d, body %s", i, statuses[i], bodies[i])
		}
		if !bytes.Equal(bodies[i], bodies[0]) {
			t.Fatalf("request %d returned a different body", i)
		}
	}
	if got := e.countEvents(t, acct); got != 1 {
		t.Errorf("usage rows after %d concurrent duplicates = %d, want 1", n, got)
	}
}

func TestKeysAreScopedPerAccount(t *testing.T) {
	e := newTestEnv(t)
	acctA, keyA := e.newAccount(t)
	acctB, keyB := e.newAccount(t)

	for _, k := range []string{keyA, keyB} {
		if resp, b := e.post(t, k, "shared-key", body); resp.StatusCode != http.StatusCreated || resp.Header.Get("Idempotent-Replayed") != "" {
			t.Fatalf("status = %d replayed=%q body %s", resp.StatusCode, resp.Header.Get("Idempotent-Replayed"), b)
		}
	}
	if a, b := e.countEvents(t, acctA), e.countEvents(t, acctB); a != 1 || b != 1 {
		t.Errorf("rows A=%d B=%d, want 1 each", a, b)
	}
}

func TestDistinctKeysCreateDistinctRows(t *testing.T) {
	e := newTestEnv(t)
	acct, key := e.newAccount(t)
	for i := range 3 {
		if resp, b := e.post(t, key, fmt.Sprintf("req-%d", i), body); resp.StatusCode != http.StatusCreated {
			t.Fatalf("status = %d, body %s", resp.StatusCode, b)
		}
	}
	if n := e.countEvents(t, acct); n != 3 {
		t.Errorf("usage rows = %d, want 3", n)
	}
}

func TestRequestErrors(t *testing.T) {
	e := newTestEnv(t)
	_, key := e.newAccount(t)

	tests := []struct {
		name, apiKey, idemKey, body string
		want                        int
	}{
		{"no api key", "", "k", body, http.StatusUnauthorized},
		{"wrong api key", "mk_test_nope", "k", body, http.StatusUnauthorized},
		{"no idempotency key", key, "", body, http.StatusBadRequest},
		{"bad body", key, "k-bad", `{"model":""}`, http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, b := e.post(t, tc.apiKey, tc.idemKey, tc.body)
			if resp.StatusCode != tc.want {
				t.Errorf("status = %d, want %d, body %s", resp.StatusCode, tc.want, b)
			}
		})
	}
}

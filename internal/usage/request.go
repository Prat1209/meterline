package usage

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

const (
	maxBodyBytes   = 64 << 10
	maxModelLen    = 100
	maxIdemKeyLen  = 255
	requestHashTag = "POST /v1/usage\n"
)

// Request is the body of POST /v1/usage. Pointer fields let us tell
// "missing" apart from zero.
type Request struct {
	Model        string     `json:"model"`
	InputTokens  *int64     `json:"input_tokens"`
	OutputTokens *int64     `json:"output_tokens"`
	OccurredAt   *time.Time `json:"occurred_at,omitempty"`
}

// Event is a validated usage record ready to store.
type Event struct {
	Model        string
	InputTokens  int64
	OutputTokens int64
	OccurredAt   time.Time
}

// Decode parses and validates a request body. Callers should cap the body
// size (the handler uses http.MaxBytesReader).
func Decode(r io.Reader) (Request, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var req Request
	if err := dec.Decode(&req); err != nil {
		return Request{}, fmt.Errorf("invalid JSON body: %w", err)
	}
	if dec.More() {
		return Request{}, errors.New("body must contain a single JSON object")
	}
	return req, req.validate()
}

func (r Request) validate() error {
	switch {
	case r.Model == "":
		return errors.New("model is required")
	case len(r.Model) > maxModelLen:
		return fmt.Errorf("model must be at most %d characters", maxModelLen)
	case r.InputTokens == nil:
		return errors.New("input_tokens is required")
	case r.OutputTokens == nil:
		return errors.New("output_tokens is required")
	case *r.InputTokens < 0 || *r.OutputTokens < 0:
		return errors.New("token counts must be non-negative")
	}
	return nil
}

// Hash fingerprints the request so a reused idempotency key with a different
// body can be detected. It hashes the re-encoded struct, not the raw bytes, so
// whitespace and JSON key order do not change the result.
func (r Request) Hash() []byte {
	canonical, _ := json.Marshal(r) // cannot fail: only plain types
	sum := sha256.Sum256(append([]byte(requestHashTag), canonical...))
	return sum[:]
}

// Event converts the request, defaulting occurred_at to now.
func (r Request) Event(now time.Time) Event {
	occurred := now
	if r.OccurredAt != nil {
		occurred = *r.OccurredAt
	}
	return Event{
		Model:        r.Model,
		InputTokens:  *r.InputTokens,
		OutputTokens: *r.OutputTokens,
		OccurredAt:   occurred.UTC(),
	}
}

// ValidIdempotencyKey reports whether k is an acceptable Idempotency-Key header.
func ValidIdempotencyKey(k string) bool {
	return k != "" && len(k) <= maxIdemKeyLen && !bytes.ContainsAny([]byte(k), "\r\n")
}

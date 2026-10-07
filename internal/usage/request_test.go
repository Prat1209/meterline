package usage

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestDecodeValidation(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr string
	}{
		{"valid", `{"model":"m","input_tokens":1,"output_tokens":2}`, ""},
		{"zero tokens ok", `{"model":"m","input_tokens":0,"output_tokens":0}`, ""},
		{"missing model", `{"input_tokens":1,"output_tokens":2}`, "model is required"},
		{"missing input", `{"model":"m","output_tokens":2}`, "input_tokens is required"},
		{"missing output", `{"model":"m","input_tokens":1}`, "output_tokens is required"},
		{"negative", `{"model":"m","input_tokens":-1,"output_tokens":2}`, "non-negative"},
		{"unknown field", `{"model":"m","input_tokens":1,"output_tokens":2,"cost":5}`, "unknown field"},
		{"float tokens", `{"model":"m","input_tokens":1.5,"output_tokens":2}`, "invalid JSON"},
		{"two objects", `{"model":"m","input_tokens":1,"output_tokens":2}{}`, "single JSON object"},
		{"model too long", `{"model":"` + strings.Repeat("x", 101) + `","input_tokens":1,"output_tokens":2}`, "at most"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Decode(strings.NewReader(tc.body))
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestHashIgnoresKeyOrderAndWhitespace(t *testing.T) {
	a, err := Decode(strings.NewReader(`{"model":"m","input_tokens":1,"output_tokens":2}`))
	if err != nil {
		t.Fatal(err)
	}
	b, err := Decode(strings.NewReader("{ \"output_tokens\": 2,\n \"input_tokens\": 1, \"model\": \"m\" }"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Hash(), b.Hash()) {
		t.Error("equivalent bodies hashed differently")
	}

	c, _ := Decode(strings.NewReader(`{"model":"m","input_tokens":1,"output_tokens":3}`))
	if bytes.Equal(a.Hash(), c.Hash()) {
		t.Error("different bodies hashed the same")
	}
}

func TestEventDefaultsOccurredAt(t *testing.T) {
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	req, _ := Decode(strings.NewReader(`{"model":"m","input_tokens":1,"output_tokens":2}`))
	if got := req.Event(now).OccurredAt; !got.Equal(now) {
		t.Errorf("OccurredAt = %v, want %v", got, now)
	}
}

func TestValidIdempotencyKey(t *testing.T) {
	for k, want := range map[string]bool{
		"":                       false,
		"abc":                    true,
		strings.Repeat("k", 255): true,
		strings.Repeat("k", 256): false,
		"a\nb":                   false,
	} {
		if got := ValidIdempotencyKey(k); got != want {
			t.Errorf("ValidIdempotencyKey(%q) = %v, want %v", k, got, want)
		}
	}
}

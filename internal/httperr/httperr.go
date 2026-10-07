// Package httperr writes JSON error responses in one consistent shape.
package httperr

import (
	"encoding/json"
	"net/http"
)

type body struct {
	Error detail `json:"error"`
}

type detail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Write sends {"error":{"type":...,"message":...}} with the given status.
func Write(w http.ResponseWriter, status int, typ, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body{Error: detail{Type: typ, Message: msg}})
}

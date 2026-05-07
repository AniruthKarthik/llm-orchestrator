package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// WriteJSON serialises v as JSON and writes it to w with the given HTTP status code. It always sets Content-Type to application/json.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		fmt.Printf("[utils] WriteJSON encode error: %v\n", err)
	}
}

// WriteError writes a structured JSON error body.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}

// DecodeJSON reads exactly one JSON value from r into dst. Returns an error if the body is empty, malformed, or contains unknown fields.
func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	return nil
}

// StrictUnmarshal unmarshals b into dst and returns an error if b is not valid JSON.
func StrictUnmarshal(b []byte, dst any) error {
	if !json.Valid(b) {
		return fmt.Errorf("response is not valid JSON")
	}
	return json.Unmarshal(b, dst)
}

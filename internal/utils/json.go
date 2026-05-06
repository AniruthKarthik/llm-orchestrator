package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// WriteJSON sends a JSON response with the specified status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		fmt.Printf("[utils] WriteJSON encode error: %v\n", err)
	}
}

// WriteError sends a JSON error response with a simple message.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}

// DecodeJSON reads the request body and decodes it into the provided destination.
func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	return nil
}

// StrictUnmarshal validates that data is proper JSON before unmarshaling it.
func StrictUnmarshal(b []byte, dst any) error {
	if !json.Valid(b) {
		return fmt.Errorf("response is not valid JSON")
	}
	return json.Unmarshal(b, dst)
}

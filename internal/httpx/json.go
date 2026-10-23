// Package httpx contains the JSON helpers shared by every handler.
package httpx

import (
	"encoding/json"
	"net/http"
)

// JSON writes v as the JSON response body with the given status code.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Error writes the standard Indonesian error envelope.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}

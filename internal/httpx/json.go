// Package httpx contains the JSON helpers shared by every handler.
package httpx

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
)

const maxBodyBytes = 1 << 20 // 1 MiB

// ErrUnsupportedMediaType is returned when a mutation is not JSON.
var ErrUnsupportedMediaType = errors.New("content type must be application/json")

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

// DecodeJSON enforces a JSON content type and decodes a single object.
func DecodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return ErrUnsupportedMediaType
	}

	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// DecodeError writes the Indonesian error envelope for a DecodeJSON failure.
func DecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrUnsupportedMediaType) {
		Error(w, http.StatusUnsupportedMediaType, "Format data harus application/json.")
		return
	}
	Error(w, http.StatusBadRequest, "Data yang dikirim tidak valid.")
}

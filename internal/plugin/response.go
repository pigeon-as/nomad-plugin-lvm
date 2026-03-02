package plugin

import (
	"encoding/json"
	"io"
)

// CreateResponse is the JSON response for a successful create operation.
type CreateResponse struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

// FingerprintResponse is the JSON response for a fingerprint operation.
type FingerprintResponse struct {
	Version string `json:"version"`
}

// WriteJSON encodes v as JSON to w.
func WriteJSON(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}

// WriteError writes a JSON error response to w.
func WriteError(w io.Writer, err error) error {
	return json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

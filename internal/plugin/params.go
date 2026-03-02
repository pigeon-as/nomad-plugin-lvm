package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// Params represents the DHV_PARAMETERS JSON payload sent by Nomad.
type Params struct {
	Type       string `json:"type"`
	Source     string `json:"source"`
	Filesystem string `json:"filesystem"`
}

// ParseParams reads and parses DHV_PARAMETERS from the environment.
func ParseParams() (*Params, error) {
	raw := os.Getenv("DHV_PARAMETERS")
	if raw == "" || raw == "{}" {
		return &Params{Type: "persistent", Filesystem: "ext4"}, nil
	}
	var p Params
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return nil, fmt.Errorf("parsing DHV_PARAMETERS: %w", err)
	}
	if p.Type == "" {
		p.Type = "persistent"
	}
	if p.Filesystem == "" {
		p.Filesystem = "ext4"
	}
	return &p, nil
}

// ParseCapacity reads DHV_CAPACITY_MIN_BYTES from the environment.
func ParseCapacity() (int64, error) {
	v := os.Getenv("DHV_CAPACITY_MIN_BYTES")
	if v == "" {
		return 0, fmt.Errorf("DHV_CAPACITY_MIN_BYTES is required for persistent volumes")
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing DHV_CAPACITY_MIN_BYTES=%q: %w", v, err)
	}
	if n <= 0 {
		return 0, fmt.Errorf("DHV_CAPACITY_MIN_BYTES must be > 0")
	}
	return n, nil
}

// RequiredEnv reads a required environment variable.
func RequiredEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return v, nil
}

// Operation reads the plugin operation from DHV_OPERATION (preferred)
// or falls back to os.Args[1].
func Operation() (string, error) {
	if op := os.Getenv("DHV_OPERATION"); op != "" {
		return op, nil
	}
	if len(os.Args) >= 2 {
		return os.Args[1], nil
	}
	return "", fmt.Errorf("no operation specified (set DHV_OPERATION or pass as argument)")
}

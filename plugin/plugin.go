// Package plugin defines the Nomad Dynamic Host Volume plugin contract.
// It contains the env var constants, request/response types, the Plugin
// interface, and helpers for parsing DHV_ environment variables.
// Implementations live elsewhere (e.g. internal/lvm).
package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
)

// Environment variables set by Nomad for external host volume plugins.
// https://github.com/hashicorp/nomad/blob/main/client/hostvolumemanager/host_volume_plugin.go
const (
	EnvOperation   = "DHV_OPERATION"
	EnvPluginDir   = "DHV_PLUGIN_DIR"
	EnvVolumeID    = "DHV_VOLUME_ID"
	EnvCapacityMin = "DHV_CAPACITY_MIN_BYTES"
	EnvParameters  = "DHV_PARAMETERS"
	EnvCreatedPath = "DHV_CREATED_PATH"
)

// Plugin is the interface that a Nomad Dynamic Host Volume plugin must implement.
type Plugin interface {
	Fingerprint() (*FingerprintResponse, error)
	Create(req *Request) (*CreateResponse, error)
	Delete(req *Request) error
}

// Request holds the environment variables Nomad passes to the plugin,
// parsed once at startup and then passed to plugin methods.
type Request struct {
	Operation   string
	VolumeID    string
	CapacityMin int64 // bytes; 0 if not set
	Parameters  Params
}

// Params represents the DHV_PARAMETERS JSON payload sent by Nomad.
type Params struct {
	Type       string `json:"type"`
	Source     string `json:"source"`
	Filesystem string `json:"filesystem"`
}

// FingerprintResponse is returned by Fingerprint.
type FingerprintResponse struct {
	Version string `json:"version"`
}

// CreateResponse is returned by Create.
type CreateResponse struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

// ParseRequest reads all DHV_ environment variables into a Request.
func ParseRequest() (*Request, error) {
	req := &Request{}

	// Operation: prefer env var, fall back to os.Args[1].
	req.Operation = os.Getenv(EnvOperation)
	if req.Operation == "" && len(os.Args) >= 2 {
		req.Operation = os.Args[1]
	}
	if req.Operation == "" {
		return nil, fmt.Errorf("no operation specified (set %s or pass as argument)", EnvOperation)
	}

	req.VolumeID = os.Getenv(EnvVolumeID)

	if v := os.Getenv(EnvCapacityMin); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing %s=%q: %w", EnvCapacityMin, v, err)
		}
		req.CapacityMin = n
	}

	params, err := parseParams(os.Getenv(EnvParameters))
	if err != nil {
		return nil, err
	}
	req.Parameters = *params

	return req, nil
}

// WriteJSON encodes v as JSON to w.
func writeJSON(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}

// WriteError writes a JSON error response to w.
func writeError(w io.Writer, err error) error {
	return json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

// Run dispatches the DHV operation from req to the given plugin
// and writes the JSON response to stdout.
func Run(p Plugin, req *Request) error {
	switch req.Operation {
	case "fingerprint":
		resp, err := p.Fingerprint()
		if err != nil {
			return fmt.Errorf("fingerprint: %w", err)
		}
		return writeJSON(os.Stdout, resp)

	case "create":
		resp, err := p.Create(req)
		if err != nil {
			writeError(os.Stdout, err)
			return err
		}
		return writeJSON(os.Stdout, resp)

	case "delete":
		if err := p.Delete(req); err != nil {
			writeError(os.Stdout, err)
			return err
		}
		return nil

	default:
		return fmt.Errorf("unknown operation: %s", req.Operation)
	}
}

func parseParams(raw string) (*Params, error) {
	if raw == "" || raw == "{}" {
		return &Params{Type: "persistent", Filesystem: "ext4"}, nil
	}
	var p Params
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", EnvParameters, err)
	}
	if p.Type == "" {
		p.Type = "persistent"
	}
	if p.Filesystem == "" {
		p.Filesystem = "ext4"
	}
	return &p, nil
}

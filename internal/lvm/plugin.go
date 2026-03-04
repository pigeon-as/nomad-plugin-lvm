package lvm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

const (
	// Version is the plugin version reported by fingerprint.
	Version = "0.1.0"

	envOperation   = "DHV_OPERATION"
	envVolumeID    = "DHV_VOLUME_ID"
	envCapacityMin = "DHV_CAPACITY_MIN_BYTES"
	envParameters  = "DHV_PARAMETERS"
)

// --- DHV protocol types ---

// Params is the DHV_PARAMETERS JSON payload sent by Nomad.
type Params struct {
	Type       string `json:"type"`
	Source     string `json:"source"`
	Filesystem string `json:"filesystem"`
	Mode       string `json:"mode"`

	VolumeGroup string `json:"volume_group"`
	ThinPool    string `json:"thin_pool"`
	MountDir    string `json:"mount_dir"`
}

// Request holds parsed DHV environment variables.
type Request struct {
	Operation   string
	VolumeID    string
	CapacityMin int64
	Params      Params
}

// FingerprintResponse is the JSON output for fingerprint.
type FingerprintResponse struct {
	Version string `json:"version"`
}

// CreateResponse is the JSON output for create.
type CreateResponse struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

// ParseRequest reads DHV_ environment variables into a Request.
func ParseRequest() (*Request, error) {
	req := &Request{}

	req.Operation = os.Getenv(envOperation)
	if req.Operation == "" && len(os.Args) >= 2 {
		req.Operation = os.Args[1]
	}
	if req.Operation == "" {
		return nil, fmt.Errorf("no operation specified (set %s or pass as argument)", envOperation)
	}

	req.VolumeID = os.Getenv(envVolumeID)

	if v := os.Getenv(envCapacityMin); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing %s=%q: %w", envCapacityMin, v, err)
		}
		req.CapacityMin = n
	}

	params, err := parseParams(os.Getenv(envParameters))
	if err != nil {
		return nil, err
	}
	req.Params = *params

	return req, nil
}

func parseParams(raw string) (*Params, error) {
	if raw == "" || raw == "{}" {
		return &Params{Type: "persistent", Filesystem: "ext4", Mode: "filesystem"}, nil
	}
	var p Params
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", envParameters, err)
	}
	if p.Type == "" {
		p.Type = "persistent"
	}
	if p.Filesystem == "" {
		p.Filesystem = "ext4"
	}
	if p.Mode == "" {
		p.Mode = "filesystem"
	}
	if p.Mode != "filesystem" && p.Mode != "block" {
		return nil, fmt.Errorf("invalid mode %q (expected filesystem or block)", p.Mode)
	}
	if p.Type != "persistent" && p.Type != "snapshot" {
		return nil, fmt.Errorf("invalid type %q (expected persistent or snapshot)", p.Type)
	}
	return &p, nil
}

// --- Config ---

// Config holds validated LVM settings extracted from Params.
type Config struct {
	VolumeGroup string
	ThinPool    string
	MountDir    string
}

func configFromParams(p *Params) (*Config, error) {
	cfg := &Config{
		VolumeGroup: p.VolumeGroup,
		ThinPool:    p.ThinPool,
		MountDir:    p.MountDir,
	}
	if cfg.VolumeGroup == "" {
		return nil, errors.New("volume_group is required in parameters")
	}
	if cfg.ThinPool == "" {
		return nil, errors.New("thin_pool is required in parameters")
	}
	if cfg.MountDir == "" {
		cfg.MountDir = "/srv/nomad-volumes"
	}
	return cfg, nil
}

func (c *Config) lvPath(name string) string {
	return fmt.Sprintf("/dev/%s/%s", c.VolumeGroup, name)
}

func (c *Config) mountPath(name string) string {
	return filepath.Join(c.MountDir, name)
}

// --- Plugin ---

// Plugin implements the Nomad DHV protocol using LVM thin provisioning.
type Plugin struct {
	lvm *Client
}

// NewPlugin creates a Plugin with the given LVM Client.
func NewPlugin(client *Client) *Plugin {
	return &Plugin{lvm: client}
}

// Run dispatches a DHV request and writes JSON output to stdout.
func (p *Plugin) Run(req *Request) error {
	switch req.Operation {
	case "fingerprint":
		return writeJSON(os.Stdout, &FingerprintResponse{Version: Version})
	case "create":
		resp, err := p.create(req)
		if err != nil {
			writeError(os.Stdout, err)
			return err
		}
		return writeJSON(os.Stdout, resp)
	case "delete":
		if err := p.delete(req); err != nil {
			writeError(os.Stdout, err)
			return err
		}
		return nil
	default:
		return fmt.Errorf("unknown operation: %s", req.Operation)
	}
}

func (p *Plugin) create(req *Request) (*CreateResponse, error) {
	if req.VolumeID == "" {
		return nil, fmt.Errorf("%s is required", envVolumeID)
	}
	if err := ValidateName(req.VolumeID); err != nil {
		return nil, err
	}

	cfg, err := configFromParams(&req.Params)
	if err != nil {
		return nil, err
	}

	switch req.Params.Type {
	case "persistent":
		if req.CapacityMin <= 0 {
			return nil, fmt.Errorf("%s is required for persistent volumes", envCapacityMin)
		}
		return p.createPersistent(cfg, req.VolumeID, req.CapacityMin, &req.Params)
	case "snapshot":
		return p.createSnapshot(cfg, req.VolumeID, &req.Params)
	default:
		return nil, fmt.Errorf("unknown volume type %q (expected persistent or snapshot)", req.Params.Type)
	}
}

func (p *Plugin) delete(req *Request) error {
	if req.VolumeID == "" {
		return fmt.Errorf("%s is required", envVolumeID)
	}
	if err := ValidateName(req.VolumeID); err != nil {
		return err
	}

	cfg, err := configFromParams(&req.Params)
	if err != nil {
		return err
	}

	mountPath := cfg.mountPath(req.VolumeID)
	_ = p.lvm.Unmount(mountPath)
	_ = os.Remove(mountPath)

	return p.lvm.Remove(cfg.VolumeGroup, req.VolumeID)
}

func (p *Plugin) createPersistent(cfg *Config, volumeID string, capacity int64, params *Params) (*CreateResponse, error) {
	vg := cfg.VolumeGroup

	if !p.lvm.Exists(vg, volumeID) {
		if err := p.lvm.CreateThin(vg, cfg.ThinPool, volumeID, capacity); err != nil {
			return nil, fmt.Errorf("lvcreate: %w", err)
		}
		if err := p.lvm.Activate(vg, volumeID); err != nil {
			_ = p.lvm.Remove(vg, volumeID)
			return nil, fmt.Errorf("lvchange activate: %w", err)
		}
		if err := p.lvm.MakeFilesystem(params.Filesystem, cfg.lvPath(volumeID)); err != nil {
			_ = p.lvm.Remove(vg, volumeID)
			return nil, fmt.Errorf("mkfs: %w", err)
		}
	} else {
		// Volume exists but may be inactive (e.g. after reboot).
		if err := p.lvm.Activate(vg, volumeID); err != nil {
			return nil, fmt.Errorf("lvchange activate: %w", err)
		}
	}

	if params.Mode == "block" {
		return p.createResponse(cfg, volumeID, params.Mode)
	}

	if err := p.mountVolume(cfg, volumeID); err != nil {
		return nil, err
	}
	return p.createResponse(cfg, volumeID, params.Mode)
}

func (p *Plugin) createSnapshot(cfg *Config, volumeID string, params *Params) (*CreateResponse, error) {
	vg := cfg.VolumeGroup

	if params.Source == "" {
		return nil, fmt.Errorf("source is required for snapshot volumes")
	}
	if err := ValidateName(params.Source); err != nil {
		return nil, err
	}
	if !p.lvm.Exists(vg, params.Source) {
		return nil, fmt.Errorf("source volume %q does not exist in VG %s", params.Source, vg)
	}

	if !p.lvm.Exists(vg, volumeID) {
		if err := p.lvm.CreateSnapshot(vg, params.Source, volumeID); err != nil {
			return nil, fmt.Errorf("lvcreate snapshot: %w", err)
		}
		if err := p.lvm.Activate(vg, volumeID); err != nil {
			_ = p.lvm.Remove(vg, volumeID)
			return nil, fmt.Errorf("lvchange activate: %w", err)
		}
	} else {
		// Snapshot exists but may be inactive.
		if err := p.lvm.Activate(vg, volumeID); err != nil {
			return nil, fmt.Errorf("lvchange activate: %w", err)
		}
	}

	if params.Mode == "block" {
		return p.createResponse(cfg, volumeID, params.Mode)
	}

	if err := p.mountVolume(cfg, volumeID); err != nil {
		return nil, err
	}
	return p.createResponse(cfg, volumeID, params.Mode)
}

func (p *Plugin) mountVolume(cfg *Config, volumeID string) error {
	devPath := cfg.lvPath(volumeID)
	mountPath := cfg.mountPath(volumeID)
	if err := os.MkdirAll(mountPath, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", mountPath, err)
	}
	_ = p.lvm.Unmount(mountPath)
	if err := p.lvm.Mount(devPath, mountPath); err != nil {
		return fmt.Errorf("mount: %w", err)
	}
	return nil
}

func (p *Plugin) createResponse(cfg *Config, volumeID, mode string) (*CreateResponse, error) {
	size, err := p.lvm.SizeBytes(cfg.VolumeGroup, volumeID)
	if err != nil {
		return nil, fmt.Errorf("getting volume size: %w", err)
	}
	resPath := cfg.mountPath(volumeID)
	if mode == "block" {
		resPath = cfg.lvPath(volumeID)
	}
	return &CreateResponse{Path: resPath, Bytes: size}, nil
}

func writeJSON(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}

func writeError(w io.Writer, err error) error {
	return json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

package lvm

import (
	"fmt"
	"os"

	"github.com/pigeon-as/nomad-plugin-lvm/plugin"
)

// Version is the plugin version reported by fingerprint.
const Version = "0.1.0"

// LVMPlugin implements plugin.Plugin using LVM thin provisioning.
// Configuration (volume group, thin pool, etc.) is read from the
// request parameters on each Create/Delete call.
type LVMPlugin struct {
	LVM *Client
}

// compile-time check that LVMPlugin implements plugin.Plugin.
var _ plugin.Plugin = (*LVMPlugin)(nil)

// NewPlugin creates an LVMPlugin with the given LVM client.
func NewPlugin(client *Client) *LVMPlugin {
	return &LVMPlugin{LVM: client}
}

// Fingerprint returns the plugin version.
func (p *LVMPlugin) Fingerprint() (*plugin.FingerprintResponse, error) {
	return &plugin.FingerprintResponse{Version: Version}, nil
}

// Create creates a new LVM volume (persistent or snapshot).
func (p *LVMPlugin) Create(req *plugin.Request) (*plugin.CreateResponse, error) {
	if req.VolumeID == "" {
		return nil, fmt.Errorf("%s is required", plugin.EnvVolumeID)
	}
	if err := ValidateName(req.VolumeID); err != nil {
		return nil, err
	}

	cfg, err := ConfigFromParams(&req.Parameters)
	if err != nil {
		return nil, err
	}

	switch req.Parameters.Type {
	case "persistent":
		if req.CapacityMin <= 0 {
			return nil, fmt.Errorf("%s is required for persistent volumes", plugin.EnvCapacityMin)
		}
		return p.createPersistent(cfg, req.VolumeID, req.CapacityMin, &req.Parameters)
	case "snapshot":
		return p.createSnapshot(cfg, req.VolumeID, &req.Parameters)
	default:
		return nil, fmt.Errorf("unknown volume type %q (expected persistent or snapshot)", req.Parameters.Type)
	}
}

// Delete removes an existing LVM volume.
func (p *LVMPlugin) Delete(req *plugin.Request) error {
	if req.VolumeID == "" {
		return fmt.Errorf("%s is required", plugin.EnvVolumeID)
	}
	if err := ValidateName(req.VolumeID); err != nil {
		return err
	}

	cfg, err := ConfigFromParams(&req.Parameters)
	if err != nil {
		return err
	}

	// Best-effort unmount before removal.
	mountPath := cfg.MountPath(req.VolumeID)
	_ = p.LVM.Unmount(mountPath)
	_ = os.Remove(mountPath)

	return p.LVM.Remove(cfg.VolumeGroup, req.VolumeID)
}

func (p *LVMPlugin) createPersistent(cfg *Config, volumeID string, capacity int64, params *plugin.Params) (*plugin.CreateResponse, error) {
	vg := cfg.VolumeGroup
	devPath := cfg.LVPath(volumeID)

	if !p.LVM.Exists(vg, volumeID) {
		if err := p.LVM.CreateThin(vg, cfg.ThinPool, volumeID, capacity); err != nil {
			return nil, fmt.Errorf("lvcreate: %w", err)
		}
		if err := p.LVM.Activate(vg, volumeID); err != nil {
			_ = p.LVM.Remove(vg, volumeID)
			return nil, fmt.Errorf("lvchange activate: %w", err)
		}
		if err := p.LVM.MakeFilesystem(params.Filesystem, devPath); err != nil {
			_ = p.LVM.Remove(vg, volumeID)
			return nil, fmt.Errorf("mkfs: %w", err)
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

func (p *LVMPlugin) createSnapshot(cfg *Config, volumeID string, params *plugin.Params) (*plugin.CreateResponse, error) {
	vg := cfg.VolumeGroup

	if params.Source == "" {
		return nil, fmt.Errorf("source is required for snapshot volumes")
	}
	if err := ValidateName(params.Source); err != nil {
		return nil, err
	}
	if !p.LVM.Exists(vg, params.Source) {
		return nil, fmt.Errorf("source volume %q does not exist in VG %s", params.Source, vg)
	}
	if !p.LVM.Exists(vg, volumeID) {
		if err := p.LVM.CreateSnapshot(vg, params.Source, volumeID); err != nil {
			return nil, fmt.Errorf("lvcreate snapshot: %w", err)
		}
		if err := p.LVM.Activate(vg, volumeID); err != nil {
			_ = p.LVM.Remove(vg, volumeID)
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

// mountVolume creates the mount directory and mounts the volume's device there.
func (p *LVMPlugin) mountVolume(cfg *Config, volumeID string) error {
	devPath := cfg.LVPath(volumeID)
	mountPath := cfg.MountPath(volumeID)
	if err := os.MkdirAll(mountPath, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", mountPath, err)
	}
	_ = p.LVM.Unmount(mountPath) // idempotent: no-op if not mounted
	if err := p.LVM.Mount(devPath, mountPath); err != nil {
		return fmt.Errorf("mount: %w", err)
	}
	return nil
}

func (p *LVMPlugin) createResponse(cfg *Config, volumeID, mode string) (*plugin.CreateResponse, error) {
	size, err := p.LVM.SizeBytes(cfg.VolumeGroup, volumeID)
	if err != nil {
		return nil, fmt.Errorf("getting volume size: %w", err)
	}
	resPath := cfg.MountPath(volumeID)
	if mode == "block" {
		resPath = cfg.LVPath(volumeID)
	}
	return &plugin.CreateResponse{Path: resPath, Bytes: size}, nil
}

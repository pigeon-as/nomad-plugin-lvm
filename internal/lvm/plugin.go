package lvm

import (
	"fmt"

	"github.com/pigeon-as/nomad-plugin-lvm/plugin"
)

// Version is the plugin version reported by fingerprint.
const Version = "0.1.0"

// LVMPlugin implements plugin.Plugin using LVM thin provisioning.
type LVMPlugin struct {
	Config *Config
	LVM    *Client
}

// compile-time check that LVMPlugin implements plugin.Plugin.
var _ plugin.Plugin = (*LVMPlugin)(nil)

// NewPlugin creates an LVMPlugin with the given config and LVM client.
func NewPlugin(cfg *Config, client *Client) *LVMPlugin {
	return &LVMPlugin{Config: cfg, LVM: client}
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

	switch req.Parameters.Type {
	case "persistent":
		if req.CapacityMin <= 0 {
			return nil, fmt.Errorf("%s is required for persistent volumes", plugin.EnvCapacityMin)
		}
		return p.createPersistent(req.VolumeID, req.CapacityMin, req.Parameters.Filesystem)
	case "snapshot":
		return p.createSnapshot(req.VolumeID, &req.Parameters)
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
	return p.LVM.Remove(p.Config.VolumeGroup, req.VolumeID)
}

func (p *LVMPlugin) createPersistent(volumeID string, capacity int64, fs string) (*plugin.CreateResponse, error) {
	vg := p.Config.VolumeGroup
	if !p.LVM.Exists(vg, volumeID) {
		if err := p.LVM.CreateThin(vg, p.Config.ThinPool, volumeID, capacity); err != nil {
			return nil, fmt.Errorf("lvcreate: %w", err)
		}
		if err := p.LVM.Activate(vg, volumeID); err != nil {
			_ = p.LVM.Remove(vg, volumeID)
			return nil, fmt.Errorf("lvchange activate: %w", err)
		}
		if err := p.LVM.MakeFilesystem(fs, p.Config.LVPath(volumeID)); err != nil {
			_ = p.LVM.Remove(vg, volumeID)
			return nil, fmt.Errorf("mkfs: %w", err)
		}
	}
	return p.createResponse(volumeID)
}

func (p *LVMPlugin) createSnapshot(volumeID string, params *plugin.Params) (*plugin.CreateResponse, error) {
	vg := p.Config.VolumeGroup
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
	return p.createResponse(volumeID)
}

func (p *LVMPlugin) createResponse(volumeID string) (*plugin.CreateResponse, error) {
	path := p.Config.LVPath(volumeID)
	size, err := p.LVM.SizeBytes(p.Config.VolumeGroup, volumeID)
	if err != nil {
		return nil, fmt.Errorf("getting volume size: %w", err)
	}
	return &plugin.CreateResponse{Path: path, Bytes: size}, nil
}

package plugin

import (
	"fmt"
	"io"
	"os"

	"github.com/pigeon-as/nomad-plugin-lvm/internal/lvm"
)

// Version is the plugin version reported by fingerprint.
const Version = "0.1.0"

// Plugin implements the Nomad Dynamic Host Volume plugin operations.
type Plugin struct {
	Config *Config
	LVM    *lvm.Client
	Stdout io.Writer
}

// New creates a Plugin with the given config and LVM client.
func New(cfg *Config, lvmClient *lvm.Client) *Plugin {
	return &Plugin{
		Config: cfg,
		LVM:    lvmClient,
		Stdout: os.Stdout,
	}
}

// Fingerprint writes plugin version information to stdout.
func (p *Plugin) Fingerprint() error {
	return WriteJSON(p.Stdout, FingerprintResponse{Version: Version})
}

// Create creates a new volume based on DHV_ environment variables.
func (p *Plugin) Create() error {
	volumeID, err := RequiredEnv("DHV_VOLUME_ID")
	if err != nil {
		return err
	}
	if err := lvm.ValidateName(volumeID); err != nil {
		return err
	}

	params, err := ParseParams()
	if err != nil {
		return err
	}

	switch params.Type {
	case "persistent":
		capacity, err := ParseCapacity()
		if err != nil {
			return err
		}
		return p.createPersistent(volumeID, capacity, params.Filesystem)
	case "snapshot":
		return p.createSnapshot(volumeID, params)
	default:
		return fmt.Errorf("unknown volume type %q (expected persistent or snapshot)", params.Type)
	}
}

// Delete removes an existing volume.
func (p *Plugin) Delete() error {
	volumeID, err := RequiredEnv("DHV_VOLUME_ID")
	if err != nil {
		return err
	}
	if err := lvm.ValidateName(volumeID); err != nil {
		return err
	}
	return p.LVM.Remove(p.Config.VolumeGroup, volumeID)
}

func (p *Plugin) createPersistent(volumeID string, capacity int64, fs string) error {
	vg := p.Config.VolumeGroup
	if !p.LVM.Exists(vg, volumeID) {
		if err := p.LVM.CreateThin(vg, p.Config.ThinPool, volumeID, capacity); err != nil {
			return fmt.Errorf("lvcreate: %w", err)
		}
		if err := p.LVM.Activate(vg, volumeID); err != nil {
			_ = p.LVM.Remove(vg, volumeID)
			return fmt.Errorf("lvchange activate: %w", err)
		}
		if err := p.LVM.MakeFilesystem(fs, p.Config.LVPath(volumeID)); err != nil {
			_ = p.LVM.Remove(vg, volumeID)
			return fmt.Errorf("mkfs: %w", err)
		}
	}
	return p.writeCreateResponse(volumeID)
}

func (p *Plugin) createSnapshot(volumeID string, params *Params) error {
	vg := p.Config.VolumeGroup
	if params.Source == "" {
		return fmt.Errorf("source is required for snapshot volumes")
	}
	if err := lvm.ValidateName(params.Source); err != nil {
		return err
	}
	if !p.LVM.Exists(vg, params.Source) {
		return fmt.Errorf("source volume %q does not exist in VG %s", params.Source, vg)
	}
	if !p.LVM.Exists(vg, volumeID) {
		if err := p.LVM.CreateSnapshot(vg, params.Source, volumeID); err != nil {
			return fmt.Errorf("lvcreate snapshot: %w", err)
		}
		if err := p.LVM.Activate(vg, volumeID); err != nil {
			_ = p.LVM.Remove(vg, volumeID)
			return fmt.Errorf("lvchange activate: %w", err)
		}
	}
	return p.writeCreateResponse(volumeID)
}

func (p *Plugin) writeCreateResponse(volumeID string) error {
	path := p.Config.LVPath(volumeID)
	size, err := p.LVM.SizeBytes(p.Config.VolumeGroup, volumeID)
	if err != nil {
		return fmt.Errorf("getting volume size: %w", err)
	}
	return WriteJSON(p.Stdout, CreateResponse{Path: path, Bytes: size})
}

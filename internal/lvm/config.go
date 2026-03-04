package lvm

import (
	"errors"
	"fmt"
	"path"

	"github.com/pigeon-as/nomad-plugin-lvm/plugin"
)

// Config holds validated LVM settings extracted from request parameters.
type Config struct {
	VolumeGroup string
	ThinPool    string
	MountDir    string
	BinPath     string
}

// ConfigFromParams builds and validates a Config from the DHV_PARAMETERS
// payload. Required fields (volume_group, thin_pool) must be set in
// the volume's parameters {} block. Optional fields fall back to
// built-in defaults.
func ConfigFromParams(p *plugin.Params) (*Config, error) {
	cfg := &Config{
		VolumeGroup: p.VolumeGroup,
		ThinPool:    p.ThinPool,
		MountDir:    p.MountDir,
		BinPath:     p.BinPath,
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.VolumeGroup == "" {
		return errors.New("volume_group is required in parameters")
	}
	if c.ThinPool == "" {
		return errors.New("thin_pool is required in parameters")
	}
	if c.MountDir == "" {
		c.MountDir = "/srv/nomad-volumes"
	}
	if c.BinPath == "" {
		c.BinPath = "/usr/sbin"
	}
	return nil
}

// LVPath returns the device path for a logical volume.
func (c *Config) LVPath(name string) string {
	return fmt.Sprintf("/dev/%s/%s", c.VolumeGroup, name)
}

// MountPath returns the mount point directory for a volume.
func (c *Config) MountPath(name string) string {
	return path.Join(c.MountDir, name)
}

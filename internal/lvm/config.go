package lvm

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const configFileName = "nomad-plugin-lvm.json"

// Config holds the LVM plugin configuration read from disk.
type Config struct {
	VolumeGroup string `json:"volume_group"`
	ThinPool    string `json:"thin_pool"`
}

// Validate checks that all required fields are present.
func (c *Config) Validate() error {
	if c.VolumeGroup == "" {
		return errors.New("volume_group is required")
	}
	if c.ThinPool == "" {
		return errors.New("thin_pool is required")
	}
	return nil
}

// LVPath returns the device path for a logical volume.
func (c *Config) LVPath(name string) string {
	return fmt.Sprintf("/dev/%s/%s", c.VolumeGroup, name)
}

// LoadConfig reads and validates the plugin config from pluginDir.
// If pluginDir is empty it falls back to the directory of the running binary.
func LoadConfig(pluginDir string) (*Config, error) {
	if pluginDir == "" {
		exe, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("cannot determine plugin directory: %w", err)
		}
		pluginDir = filepath.Dir(exe)
	}

	path := filepath.Join(pluginDir, configFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}

	return &cfg, nil
}

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

const (
	pluginVersion  = "0.1.0"
	configFileName = "nomad-plugin-lvm.json"
)

type Config struct {
	VolumeGroup string `json:"volume_group"`
	ThinPool    string `json:"thin_pool"`
}

func (c *Config) Validate() error {
	if c.VolumeGroup == "" {
		return errors.New("volume_group is required")
	}
	if c.ThinPool == "" {
		return errors.New("thin_pool is required")
	}
	return nil
}

func (c *Config) LVPath(name string) string {
	return fmt.Sprintf("/dev/%s/%s", c.VolumeGroup, name)
}

func loadConfig() (*Config, error) {
	pluginDir := os.Getenv("DHV_PLUGIN_DIR")
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

type Params struct {
	Type       string `json:"type"`
	Source     string `json:"source"`
	Filesystem string `json:"filesystem"`
}

func parseParams() (*Params, error) {
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

func envRequired(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return v, nil
}

var validLVNameRe = regexp.MustCompile(`^[a-zA-Z0-9+_.\-]+$`)

func validLVName(name string) error {
	if !validLVNameRe.MatchString(name) {
		return fmt.Errorf("invalid LV name: %q", name)
	}
	return nil
}

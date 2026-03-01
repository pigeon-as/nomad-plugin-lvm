package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

func cmdCreate(cfg *Config) error {
	volumeID, err := envRequired("DHV_VOLUME_ID")
	if err != nil {
		return err
	}
	if err := validLVName(volumeID); err != nil {
		return err
	}

	params, err := parseParams()
	if err != nil {
		return err
	}

	switch params.Type {
	case "persistent":
		return createPersistent(cfg, volumeID, params)
	case "snapshot":
		return createSnapshot(cfg, volumeID, params)
	default:
		return fmt.Errorf("unknown volume type %q (expected persistent or snapshot)", params.Type)
	}
}

func createPersistent(cfg *Config, volumeID string, params *Params) error {
	v := os.Getenv("DHV_CAPACITY_MIN_BYTES")
	if v == "" {
		return fmt.Errorf("capacity_min is required for persistent volumes")
	}
	capacity, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fmt.Errorf("parsing DHV_CAPACITY_MIN_BYTES=%q: %w", v, err)
	}
	if capacity <= 0 {
		return fmt.Errorf("capacity_min must be greater than zero for persistent volumes")
	}

	if !lvExists(cfg.VolumeGroup, volumeID) {
		if err := lvCreateThin(cfg.VolumeGroup, cfg.ThinPool, volumeID, capacity); err != nil {
			return fmt.Errorf("lvcreate: %w", err)
		}
		if err := lvActivate(cfg.VolumeGroup, volumeID); err != nil {
			_ = lvRemove(cfg.VolumeGroup, volumeID)
			return fmt.Errorf("lvchange activate: %w", err)
		}
		if err := mkfs(params.Filesystem, cfg.LVPath(volumeID)); err != nil {
			_ = lvRemove(cfg.VolumeGroup, volumeID)
			return fmt.Errorf("mkfs: %w", err)
		}
	}

	return writeCreateResponse(cfg, volumeID)
}

func createSnapshot(cfg *Config, volumeID string, params *Params) error {
	if params.Source == "" {
		return fmt.Errorf("source is required for snapshot volumes")
	}
	if err := validLVName(params.Source); err != nil {
		return err
	}
	if !lvExists(cfg.VolumeGroup, params.Source) {
		return fmt.Errorf("source volume %q does not exist in VG %s", params.Source, cfg.VolumeGroup)
	}

	if !lvExists(cfg.VolumeGroup, volumeID) {
		if err := lvCreateSnapshot(cfg.VolumeGroup, params.Source, volumeID); err != nil {
			return fmt.Errorf("lvcreate snapshot: %w", err)
		}
		if err := lvActivate(cfg.VolumeGroup, volumeID); err != nil {
			_ = lvRemove(cfg.VolumeGroup, volumeID)
			return fmt.Errorf("lvchange activate: %w", err)
		}
	}

	return writeCreateResponse(cfg, volumeID)
}

func writeCreateResponse(cfg *Config, volumeID string) error {
	path := cfg.LVPath(volumeID)
	size, err := lvSizeBytes(cfg.VolumeGroup, volumeID)
	if err != nil {
		return fmt.Errorf("getting volume size: %w", err)
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{
		"path":  path,
		"bytes": size,
	})
}

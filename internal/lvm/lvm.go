package lvm

import (
	"fmt"
	"strings"
)

// Client wraps LVM command-line operations.
type Client struct {
	exec Exec
}

// New creates a Client that uses the given Exec to run commands.
func New(exec Exec) *Client {
	return &Client{exec: exec}
}

// Exists checks whether a logical volume exists.
func (c *Client) Exists(vg, lv string) bool {
	return c.exec.Run("lvs", "--noheadings", "--nosuffix", fmt.Sprintf("%s/%s", vg, lv)) == nil
}

// CreateThin creates a thin-provisioned logical volume.
func (c *Client) CreateThin(vg, thinPool, name string, sizeBytes int64) error {
	size := fmt.Sprintf("%db", sizeBytes)
	return c.exec.Run("lvcreate",
		"--thin",
		"--virtualsize", size,
		"--thinpool", thinPool,
		"--name", name,
		vg,
	)
}

// CreateSnapshot creates an LVM snapshot of an existing volume.
func (c *Client) CreateSnapshot(vg, source, name string) error {
	origin := fmt.Sprintf("%s/%s", vg, source)
	return c.exec.Run("lvcreate",
		"--snapshot",
		"--name", name,
		"--setactivationskip", "n",
		origin,
	)
}

// Remove deletes a logical volume. It is a no-op if the volume does not exist.
func (c *Client) Remove(vg, name string) error {
	if !c.Exists(vg, name) {
		return nil
	}
	return c.exec.Run("lvremove", "--force", fmt.Sprintf("%s/%s", vg, name))
}

// Activate activates a logical volume.
func (c *Client) Activate(vg, name string) error {
	return c.exec.Run("lvchange", "--activate", "y", fmt.Sprintf("%s/%s", vg, name))
}

// SizeBytes returns the size of a logical volume in bytes.
func (c *Client) SizeBytes(vg, name string) (int64, error) {
	out, err := c.exec.Output("lvs",
		"--noheadings", "--nosuffix", "--units", "b",
		"--options", "lv_size",
		fmt.Sprintf("%s/%s", vg, name),
	)
	if err != nil {
		return 0, err
	}
	var size int64
	if _, err := fmt.Sscan(strings.TrimSpace(out), &size); err != nil {
		return 0, fmt.Errorf("parsing lv_size %q: %w", out, err)
	}
	return size, nil
}

// MakeFilesystem creates a filesystem on the given device.
func (c *Client) MakeFilesystem(fsType, device string) error {
	if fsType != "ext4" {
		return fmt.Errorf("unsupported filesystem type: %q", fsType)
	}
	return c.exec.Run("mkfs.ext4", "-q", device)
}

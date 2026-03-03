package lvm

import (
	"fmt"
	"path"
	"strings"
)

// Client wraps LVM command-line operations.
// Binary paths are resolved once at construction time, following the
// terraform-exec pattern of storing absolute paths as struct fields.
type Client struct {
	exec Exec

	lvs      string // absolute path to lvs
	lvcreate string // absolute path to lvcreate
	lvremove string // absolute path to lvremove
	lvchange string // absolute path to lvchange
	mkfsExt4 string // absolute path to mkfs.ext4
	mount    string // absolute path to mount
	umount   string // absolute path to umount
}

// New creates a Client that uses the given Exec to run commands.
// binPath is the directory containing LVM and mkfs binaries (e.g. /usr/sbin).
func New(exec Exec, binPath string) *Client {
	bin := func(name string) string { return path.Join(binPath, name) }
	return &Client{
		exec:     exec,
		lvs:      bin("lvs"),
		lvcreate: bin("lvcreate"),
		lvremove: bin("lvremove"),
		lvchange: bin("lvchange"),
		mkfsExt4: bin("mkfs.ext4"),
		mount:    "/usr/bin/mount",
		umount:   "/usr/bin/umount",
	}
}

// Exists checks whether a logical volume exists.
func (c *Client) Exists(vg, lv string) bool {
	return c.exec.Run(c.lvs, "--noheadings", "--nosuffix", fmt.Sprintf("%s/%s", vg, lv)) == nil
}

// CreateThin creates a thin-provisioned logical volume.
func (c *Client) CreateThin(vg, thinPool, name string, sizeBytes int64) error {
	size := fmt.Sprintf("%db", sizeBytes)
	return c.exec.Run(c.lvcreate,
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
	return c.exec.Run(c.lvcreate,
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
	return c.exec.Run(c.lvremove, "--force", fmt.Sprintf("%s/%s", vg, name))
}

// Activate activates a logical volume.
func (c *Client) Activate(vg, name string) error {
	return c.exec.Run(c.lvchange, "--activate", "y", fmt.Sprintf("%s/%s", vg, name))
}

// SizeBytes returns the size of a logical volume in bytes.
func (c *Client) SizeBytes(vg, name string) (int64, error) {
	out, err := c.exec.Output(c.lvs,
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
	return c.exec.Run(c.mkfsExt4, "-q", device)
}

// Mount mounts a device at the given target directory.
func (c *Client) Mount(device, target string) error {
	return c.exec.Run(c.mount, device, target)
}

// Unmount unmounts the given mount point, ignoring errors if not mounted.
func (c *Client) Unmount(target string) error {
	return c.exec.Run(c.umount, target)
}

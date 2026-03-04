package lvm

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Exec runs system commands.
type Exec interface {
	Run(name string, args ...string) error
	Output(name string, args ...string) (string, error)
}

// ExecCommand executes commands via os/exec.
type ExecCommand struct{}

func (e ExecCommand) Run(name string, args ...string) error {
	_, err := e.Output(name, args...)
	return err
}

func (ExecCommand) Output(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

var validNameRe = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.-]*$`)

// ValidateName checks whether a string is a valid LVM logical volume name.
func ValidateName(name string) error {
	if !validNameRe.MatchString(name) {
		return fmt.Errorf("invalid LV name: %q", name)
	}
	return nil
}

// Client wraps LVM command-line operations.
type Client struct {
	exec    Exec
	binPath string
}

func (c *Client) bin(name string) string {
	return filepath.Join(c.binPath, name)
}

// NewClient creates a Client that resolves binaries from binPath.
func NewClient(exec Exec, binPath string) *Client {
	return &Client{exec: exec, binPath: binPath}
}

// Exists checks whether a logical volume exists.
func (c *Client) Exists(vg, lv string) bool {
	return c.exec.Run(c.bin("lvs"), "--noheadings", "--nosuffix", fmt.Sprintf("%s/%s", vg, lv)) == nil
}

// CreateThin creates a thin-provisioned logical volume.
func (c *Client) CreateThin(vg, thinPool, name string, sizeBytes int64) error {
	size := fmt.Sprintf("%db", sizeBytes)
	return c.exec.Run(c.bin("lvcreate"),
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
	return c.exec.Run(c.bin("lvcreate"),
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
	return c.exec.Run(c.bin("lvremove"), "--force", fmt.Sprintf("%s/%s", vg, name))
}

// Activate activates a logical volume.
func (c *Client) Activate(vg, name string) error {
	return c.exec.Run(c.bin("lvchange"), "--activate", "y", fmt.Sprintf("%s/%s", vg, name))
}

// SizeBytes returns the size of a logical volume in bytes.
func (c *Client) SizeBytes(vg, name string) (int64, error) {
	out, err := c.exec.Output(c.bin("lvs"),
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
	return c.exec.Run(c.bin("mkfs.ext4"), "-q", device)
}

// Mount mounts a device at the given target directory.
func (c *Client) Mount(device, target string) error {
	return c.exec.Run(c.bin("mount"), device, target)
}

// Unmount unmounts the given mount point.
func (c *Client) Unmount(target string) error {
	return c.exec.Run(c.bin("umount"), target)
}

package lvm

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Runner executes system commands.
type Runner interface {
	Run(name string, args ...string) error
	Output(name string, args ...string) (string, error)
}

// SystemRunner executes commands via os/exec.
type SystemRunner struct{}

func (SystemRunner) Run(name string, args ...string) error {
	_, err := SystemRunner{}.Output(name, args...)
	return err
}

func (SystemRunner) Output(name string, args ...string) (string, error) {
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

// Client wraps LVM command-line operations.
type Client struct {
	runner Runner
}

// New creates a Client that uses the given Runner to execute commands.
func New(runner Runner) *Client {
	return &Client{runner: runner}
}

// Exists checks whether a logical volume exists.
func (c *Client) Exists(vg, lv string) bool {
	return c.runner.Run("lvs", "--noheadings", "--nosuffix", fmt.Sprintf("%s/%s", vg, lv)) == nil
}

// CreateThin creates a thin-provisioned logical volume.
func (c *Client) CreateThin(vg, thinPool, name string, sizeBytes int64) error {
	size := fmt.Sprintf("%db", sizeBytes)
	return c.runner.Run("lvcreate",
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
	return c.runner.Run("lvcreate",
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
	return c.runner.Run("lvremove", "--force", fmt.Sprintf("%s/%s", vg, name))
}

// Activate activates a logical volume.
func (c *Client) Activate(vg, name string) error {
	return c.runner.Run("lvchange", "--activate", "y", fmt.Sprintf("%s/%s", vg, name))
}

// SizeBytes returns the size of a logical volume in bytes.
func (c *Client) SizeBytes(vg, name string) (int64, error) {
	out, err := c.runner.Output("lvs",
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
	return c.runner.Run("mkfs.ext4", "-q", device)
}

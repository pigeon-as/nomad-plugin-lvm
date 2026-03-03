// Copyright Pigeon AS 2025
// SPDX-License-Identifier: MPL-2.0

//go:build e2e

// Run: make e2e (requires a running nomad dev agent: make dev)

package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/shoenig/test/must"
)

// Must match e2e/nomad-plugin-lvm.json.
const (
	vg       = "e2e-vg"
	pool     = "thinpool"
	mountDir = "/tmp/nomad-volumes"
)

var (
	loopFile string
	loopDev  string
)

var volIDRe = regexp.MustCompile(`(?i)with ID\s+(\S+)`)

// --- LVM loopback infra ---

func TestMain(m *testing.M) {
	setupLVM()
	code := m.Run()
	cleanupLVM()
	os.Exit(code)
}

func setupLVM() {
	// Tear down any leftovers from a previous failed run.
	cleanupLVM()

	f, _ := os.CreateTemp("", "lvm-e2e-*.img")
	loopFile = f.Name()
	f.Close()

	mustShell("truncate", "-s", "1G", loopFile)
	loopDev = mustShell("losetup", "--find", "--show", loopFile)
	mustShell("pvcreate", loopDev)
	mustShell("vgcreate", vg, loopDev)
	mustShell("lvcreate", "--type", "thin-pool", "--name", pool, "--size", "900M", vg)
}

func cleanupLVM() {
	destroyVG(vg)

	dev := loopDev
	if dev == "" {
		if out, err := shell("pvs", "--noheadings", "-o", "pv_name", "-S", "vg_name="+vg); err == nil {
			dev = strings.TrimSpace(out)
		}
	}
	if dev != "" {
		shell("pvremove", "--force", dev)
		shell("losetup", "--detach", dev)
	}
	if loopFile != "" {
		os.Remove(loopFile)
	}
}

// destroyVG tears down a volume group no matter what state it's in.
// Device-mapper doubles hyphens in VG names: "e2e-vg" → "e2e--vg".
func destroyVG(name string) {
	dmPrefix := strings.ReplaceAll(name, "-", "--")

	if out, _ := shell("dmsetup", "ls"); out != "" {
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(line, dmPrefix+"-") {
				dm := strings.Fields(line)[0]
				shell("fuser", "-km", "/dev/mapper/"+dm)
			}
		}
	}

	if mounts, err := os.ReadFile("/proc/mounts"); err == nil {
		for _, line := range strings.Split(string(mounts), "\n") {
			if strings.Contains(line, "/"+name+"/") || strings.Contains(line, dmPrefix+"-") {
				if f := strings.Fields(line); len(f) >= 2 {
					shell("umount", "-l", f[1])
				}
			}
		}
	}

	shell("vgchange", "-an", name)
	shell("lvremove", "--force", name)
	shell("vgremove", "--force", name)

	if out, _ := shell("dmsetup", "ls"); out != "" {
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(line, dmPrefix+"-") {
				dm := strings.Fields(line)[0]
				shell("dmsetup", "remove", "--force", dm)
			}
		}
	}

	os.RemoveAll("/dev/" + name)
}

func shell(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func mustShell(name string, args ...string) string {
	out, err := shell(name, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %s: %v\n%s\n", name, strings.Join(args, " "), err, out)
		os.Exit(1)
	}
	return out
}

// --- helpers ---

func setup(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func run(t *testing.T, ctx context.Context, name string, args ...string) string {
	t.Helper()
	t.Logf("RUN '%s %s'", name, strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, name, args...)
	b, err := cmd.CombinedOutput()
	out := strings.TrimSpace(string(b))
	if err != nil {
		t.Fatalf("'%s %s' failed: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	return out
}

func tryRun(ctx context.Context, name string, args ...string) {
	exec.CommandContext(ctx, name, args...).CombinedOutput() //nolint:errcheck
}

func createVolume(t *testing.T, ctx context.Context, spec string) string {
	t.Helper()
	out := run(t, ctx, "nomad", "volume", "create", spec)
	m := volIDRe.FindStringSubmatch(out)
	if len(m) < 2 {
		t.Fatalf("could not parse volume ID from:\n%s", out)
	}
	return m[1]
}

func deleteVolume(ctx context.Context, id string) {
	tryRun(ctx, "nomad", "volume", "delete", "-type", "host", id)
}

func lvExists(volID string) bool {
	_, err := shell("lvs", "--noheadings", vg+"/"+volID)
	return err == nil
}

func snapshotSpec(t *testing.T, sourceVolID string) string {
	t.Helper()
	spec := fmt.Sprintf(`name      = "test-snapshot"
type      = "host"
plugin_id = "nomad-plugin-lvm"

capability {
  access_mode     = "single-node-writer"
  attachment_mode = "file-system"
}

parameters {
  type   = "snapshot"
  source = %q
}
`, sourceVolID)
	path := filepath.Join(t.TempDir(), "snapshot.hcl")
	must.NoError(t, os.WriteFile(path, []byte(spec), 0644))
	return path
}

// --- tests ---

// TestPersistentVolume creates a persistent volume through Nomad and
// verifies the LV exists and is mounted.
func TestPersistentVolume(t *testing.T) {
	ctx := setup(t)

	volID := createVolume(t, ctx, "./volumes/persistent.hcl")
	t.Cleanup(func() { deleteVolume(ctx, volID) })

	// LV exists in VG.
	must.True(t, lvExists(volID))

	// Mounted and writable.
	path := mountDir + "/" + volID
	must.NoError(t, os.WriteFile(filepath.Join(path, "test.txt"), []byte("hello"), 0644))

	// Nomad knows about it.
	out := run(t, ctx, "nomad", "volume", "status", "-type", "host", volID)
	must.StrContains(t, out, "test-persistent")
}

// TestSnapshotVolume creates a golden volume, snapshots it through Nomad,
// and verifies the snapshot contains the golden data.
func TestSnapshotVolume(t *testing.T) {
	ctx := setup(t)

	// 1. Create golden volume and write data to it.
	goldenVolID := createVolume(t, ctx, "./volumes/golden.hcl")
	t.Cleanup(func() { deleteVolume(ctx, goldenVolID) })

	goldenPath := mountDir + "/" + goldenVolID
	must.NoError(t, os.WriteFile(filepath.Join(goldenPath, "golden.txt"), []byte("golden-data\n"), 0644))

	// 2. Snapshot the golden volume.
	snapSpec := snapshotSpec(t, goldenVolID)
	snapVolID := createVolume(t, ctx, snapSpec)
	t.Cleanup(func() { deleteVolume(ctx, snapVolID) })

	// 3. Verify snapshot has the golden data.
	snapPath := mountDir + "/" + snapVolID
	data, err := os.ReadFile(filepath.Join(snapPath, "golden.txt"))
	must.NoError(t, err)
	must.Eq(t, "golden-data\n", string(data))

	// 4. Snapshot is independently writable.
	must.NoError(t, os.WriteFile(filepath.Join(snapPath, "snap.txt"), []byte("snap"), 0644))
}

// TestVolumeLifecycle verifies the full create → delete cycle.
func TestVolumeLifecycle(t *testing.T) {
	ctx := setup(t)

	// Create.
	volID := createVolume(t, ctx, "./volumes/lifecycle.hcl")

	// Exists.
	must.True(t, lvExists(volID))
	out := run(t, ctx, "nomad", "volume", "status", "-type", "host", volID)
	must.StrContains(t, out, "test-lifecycle")

	// Delete.
	run(t, ctx, "nomad", "volume", "delete", "-type", "host", volID)

	// Gone.
	must.False(t, lvExists(volID))
	cmd := exec.CommandContext(ctx, "nomad", "volume", "status", "-type", "host", volID)
	_, err := cmd.CombinedOutput()
	must.Error(t, err)
}

// Copyright Pigeon AS 2025
// SPDX-License-Identifier: MPL-2.0

//go:build e2e

// Run: sudo make e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shoenig/test/must"
)

const (
	vg   = "e2evg"
	pool = "e2epool"
)

var (
	pluginBin string
	configDir string
	loopFile  string
	loopDev   string
)

func TestMain(m *testing.M) {
	pluginBin, _ = filepath.Abs("../build/nomad-plugin-lvm")
	setup()
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func setup() {
	// Tear down any leftovers from a previous failed run.
	cleanup()

	// Write plugin config to a temp dir.
	configDir, _ = os.MkdirTemp("", "lvm-e2e-*")
	cfg, _ := json.Marshal(map[string]string{"volume_group": vg, "thin_pool": pool})
	os.WriteFile(filepath.Join(configDir, "nomad-plugin-lvm.json"), cfg, 0644)

	// Create a loopback thin pool.
	f, _ := os.CreateTemp("", "lvm-e2e-*.img")
	loopFile = f.Name()
	f.Close()

	mustRun(run("truncate", "-s", "200M", loopFile))
	loopDev = strings.TrimSpace(mustRun(run("losetup", "--find", "--show", loopFile)))
	mustRun(run("pvcreate", loopDev))
	mustRun(run("vgcreate", vg, loopDev))
	mustRun(run("lvcreate", "--type", "thin-pool", "--name", pool, "--size", "180M", vg))
}

func cleanup() {
	run("lvremove", "--force", vg)
	run("vgremove", "--force", vg)
	run("pvremove", "--force", loopDev)
	run("losetup", "--detach", loopDev)
	os.Remove(loopFile)
	os.RemoveAll(configDir)
}

// run executes a command and returns its output.
func run(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func mustRun(s string, err error) string {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	return s
}

// plugin runs the plugin binary and parses JSON output.
func plugin(t *testing.T, op string, env ...string) map[string]any {
	t.Helper()
	out := pluginRaw(t, op, env...)
	var resp map[string]any
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("bad JSON from %s: %v\n%s", op, err, out)
	}
	return resp
}

// pluginRaw runs the plugin binary and returns raw output.
func pluginRaw(t *testing.T, op string, env ...string) []byte {
	t.Helper()
	cmd := exec.Command(pluginBin, op)
	cmd.Env = append(os.Environ(), "DHV_PLUGIN_DIR="+configDir)
	cmd.Env = append(cmd.Env, env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s: %v\n%s", op, err, out)
	}
	return out
}

func lvExists(name string) bool {
	_, err := run("lvs", "--noheadings", vg+"/"+name)
	return err == nil
}

// --- tests ---

func TestFingerprint(t *testing.T) {
	resp := plugin(t, "fingerprint")
	must.NotEq(t, nil, resp["version"])
}

func TestPersistent(t *testing.T) {
	id := "e2e-persistent"

	// Create 10MB persistent volume.
	resp := plugin(t, "create", "DHV_VOLUME_ID="+id, "DHV_CAPACITY_MIN_BYTES=10485760")
	path := resp["path"].(string)

	// Verify ext4.
	fstype, _ := run("blkid", "-o", "value", "-s", "TYPE", path)
	must.Eq(t, "ext4", fstype)

	// Delete and verify gone.
	pluginRaw(t, "delete", "DHV_VOLUME_ID="+id)
	must.False(t, lvExists(id))
}

func TestSnapshot(t *testing.T) {
	golden := "e2e-golden"
	snap := "e2e-snap"

	// Create read-only golden volume (simulates pigeon-build).
	mustRun(run("lvcreate", "--thin", "--virtualsize", "10M", "--thinpool", pool, "--name", golden, vg))
	mustRun(run("mkfs.ext4", "-q", fmt.Sprintf("/dev/%s/%s", vg, golden)))
	mustRun(run("lvchange", "--permission", "r", vg+"/"+golden))
	t.Cleanup(func() { run("lvremove", "--force", vg+"/"+golden) })

	// Snapshot via plugin.
	params, _ := json.Marshal(map[string]string{"type": "snapshot", "source": golden})
	resp := plugin(t, "create", "DHV_VOLUME_ID="+snap, "DHV_PARAMETERS="+string(params))
	path := resp["path"].(string)

	// Verify writable: mount, write, unmount.
	mnt, _ := os.MkdirTemp("", "lvm-e2e-mnt-*")
	defer os.RemoveAll(mnt)
	mustRun(run("mount", path, mnt))
	os.WriteFile(filepath.Join(mnt, "test"), []byte("hello"), 0644)
	mustRun(run("umount", mnt))

	// Delete and verify gone.
	pluginRaw(t, "delete", "DHV_VOLUME_ID="+snap)
	must.False(t, lvExists(snap))
}

func TestDeleteIdempotent(t *testing.T) {
	pluginRaw(t, "delete", "DHV_VOLUME_ID=e2e-nonexistent")
}

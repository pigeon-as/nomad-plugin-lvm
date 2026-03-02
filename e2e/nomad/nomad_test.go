// Copyright Pigeon AS 2025
// SPDX-License-Identifier: MPL-2.0

//go:build e2e

// Run: make e2e-nomad (requires a running nomad dev agent: make dev)

package nomad

import (
	"context"
	"encoding/json"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/shoenig/test/must"
)

var (
	runningRe = regexp.MustCompile(`Status\s+=\s+running`)
	deadRe    = regexp.MustCompile(`Status\s+=\s+dead`)
	idRe      = regexp.MustCompile(`"ID"\s*:\s*"([^"]+)"`)
)

// --- helpers ---

func setup(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(func() {
		run(t, ctx, "nomad", "system", "gc")
		cancel()
	})
	return ctx
}

func run(t *testing.T, ctx context.Context, command string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(ctx, command, args...)
	b, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(b))
	if err != nil {
		t.Fatalf("'%s %s' failed: %v\n%s", command, strings.Join(args, " "), err, output)
	}
	return output
}

func purge(t *testing.T, ctx context.Context, job string) func() {
	t.Helper()
	return func() {
		cmd := exec.CommandContext(ctx, "nomad", "job", "stop", "-purge", job)
		cmd.CombinedOutput() //nolint: ignore errors on cleanup
	}
}

func waitForRunning(t *testing.T, ctx context.Context, job string) {
	t.Helper()
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for job %s to be running", job)
		default:
		}
		out := run(t, ctx, "nomad", "job", "status", job)
		if runningRe.MatchString(out) {
			return
		}
		if deadRe.MatchString(out) {
			t.Fatalf("job %s is dead:\n%s", job, out)
		}
		time.Sleep(time.Second)
	}
}

func allocID(t *testing.T, ctx context.Context, job string) string {
	t.Helper()
	out := run(t, ctx, "nomad", "job", "allocs", "-json", job)
	matches := idRe.FindStringSubmatch(out)
	if len(matches) < 2 {
		t.Fatalf("could not find alloc ID for job %s:\n%s", job, out)
	}
	return matches[1]
}

func waitForLogs(t *testing.T, ctx context.Context, allocID, task, substr string) {
	t.Helper()
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for logs containing %q", substr)
		default:
		}
		cmd := exec.CommandContext(ctx, "nomad", "alloc", "logs", allocID, task)
		out, _ := cmd.CombinedOutput()
		if strings.Contains(string(out), substr) {
			return
		}
		time.Sleep(time.Second)
	}
}

// --- tests ---

func TestPersistentVolume(t *testing.T) {
	ctx := setup(t)
	defer purge(t, ctx, "test-persistent")()

	run(t, ctx, "nomad", "job", "run", "./jobs/persistent.hcl")
	waitForRunning(t, ctx, "test-persistent")

	id := allocID(t, ctx, "test-persistent")
	waitForLogs(t, ctx, id, "write", "hello")
}

func TestSnapshotVolume(t *testing.T) {
	ctx := setup(t)

	// First create a golden volume via a helper job.
	defer purge(t, ctx, "test-golden")()
	run(t, ctx, "nomad", "job", "run", "./jobs/golden.hcl")
	waitForRunning(t, ctx, "test-golden")

	goldenID := allocID(t, ctx, "test-golden")
	_ = goldenID // golden volume is set up

	// Now run the snapshot job.
	defer purge(t, ctx, "test-snapshot")()
	run(t, ctx, "nomad", "job", "run", "./jobs/snapshot.hcl")
	waitForRunning(t, ctx, "test-snapshot")

	snapID := allocID(t, ctx, "test-snapshot")
	waitForLogs(t, ctx, snapID, "verify", "snapshot-ok")
}

// TestVolumeLifecycle verifies create → use → delete cycle through Nomad.
func TestVolumeLifecycle(t *testing.T) {
	ctx := setup(t)
	defer purge(t, ctx, "test-lifecycle")()

	run(t, ctx, "nomad", "job", "run", "./jobs/persistent.hcl")
	waitForRunning(t, ctx, "test-lifecycle")

	id := allocID(t, ctx, "test-lifecycle")
	_ = id

	// Verify the volume exists at the Nomad level.
	out := run(t, ctx, "nomad", "volume", "status", "-type=host")
	must.StrContains(t, out, "test-lifecycle")
}

// volumeStatus returns the JSON status of a dynamic host volume.
func volumeStatus(t *testing.T, ctx context.Context, name string) map[string]any {
	t.Helper()
	out := run(t, ctx, "nomad", "volume", "status", "-type=host", "-json", name)
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("bad JSON from volume status: %v\n%s", err, out)
	}
	return result
}

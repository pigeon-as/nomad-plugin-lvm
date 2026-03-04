package lvm

import (
	"fmt"
	"testing"

	"github.com/pigeon-as/nomad-plugin-lvm/plugin"
	"github.com/shoenig/test/must"
)

func testPlugin(m *mockExec) *LVMPlugin {
	return NewPlugin(New(m, testBinPath))
}

func testRequest(op, volID string, capMin int64, params plugin.Params) *plugin.Request {
	return &plugin.Request{
		Operation:   op,
		VolumeID:    volID,
		CapacityMin: capMin,
		Parameters:  params,
	}
}

func persistentParams(mountDir string) plugin.Params {
	return plugin.Params{
		Type:        "persistent",
		Filesystem:  "ext4",
		Mode:        "filesystem",
		VolumeGroup: "testvg",
		ThinPool:    "thinpool",
		MountDir:    mountDir,
	}
}

func TestFingerprint(t *testing.T) {
	p := testPlugin(newMockExec())
	resp, err := p.Fingerprint()
	must.NoError(t, err)
	must.Eq(t, Version, resp.Version)
}

func TestCreate_Persistent(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)
	mountDir := t.TempDir()

	// lvs returns error → volume doesn't exist
	m.errors["/usr/sbin/lvs --noheadings --nosuffix testvg/vol1"] = fmt.Errorf("not found")

	// SizeBytes returns 1GiB
	m.outputs["/usr/sbin/lvs --noheadings --nosuffix --units b --options lv_size testvg/vol1"] = "  1073741824"

	req := testRequest("create", "vol1", 1073741824, persistentParams(mountDir))
	resp, err := p.Create(req)
	must.NoError(t, err)
	must.Eq(t, int64(1073741824), resp.Bytes)
	must.StrContains(t, resp.Path, "vol1")

	// Verify the command sequence: lvs(exists), lvcreate, lvchange, mkfs, mkdir, umount, mount
	must.SliceLen(t, 7, m.commands)
}

func TestCreate_Persistent_RollbackOnActivateFail(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	m.errors["/usr/sbin/lvs --noheadings --nosuffix testvg/vol1"] = fmt.Errorf("not found")
	m.errors["/usr/sbin/lvchange --activate y testvg/vol1"] = fmt.Errorf("activate failed")

	req := testRequest("create", "vol1", 1073741824, persistentParams(t.TempDir()))
	_, err := p.Create(req)
	must.ErrorContains(t, err, "lvchange activate")
}

func TestCreate_Persistent_Idempotent(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	// lvs succeeds → volume already exists
	// (no error entry = success by default)

	// SizeBytes
	m.outputs["/usr/sbin/lvs --noheadings --nosuffix --units b --options lv_size testvg/vol1"] = "  1073741824"

	req := testRequest("create", "vol1", 1073741824, persistentParams(t.TempDir()))
	resp, err := p.Create(req)
	must.NoError(t, err)
	must.Eq(t, int64(1073741824), resp.Bytes)

	// Should NOT have called lvcreate (volume existed)
	for _, cmd := range m.commands {
		must.StrNotContains(t, cmd, "lvcreate")
	}
}

func TestCreate_Persistent_BlockMode(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	m.errors["/usr/sbin/lvs --noheadings --nosuffix testvg/vol1"] = fmt.Errorf("not found")
	m.outputs["/usr/sbin/lvs --noheadings --nosuffix --units b --options lv_size testvg/vol1"] = "  1073741824"

	params := persistentParams(t.TempDir())
	params.Mode = "block"
	req := testRequest("create", "vol1", 1073741824, params)
	resp, err := p.Create(req)
	must.NoError(t, err)
	must.StrContains(t, resp.Path, "/dev/testvg/vol1")

	// No mount or umount in block mode
	for _, cmd := range m.commands {
		must.StrNotContains(t, cmd, "mount")
	}
}

func TestCreate_Snapshot(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)
	mountDir := t.TempDir()

	// source exists
	// (no error for lvs testvg/source1 = exists)

	// snapshot doesn't exist
	m.errors["/usr/sbin/lvs --noheadings --nosuffix testvg/snap1"] = fmt.Errorf("not found")

	// SizeBytes
	m.outputs["/usr/sbin/lvs --noheadings --nosuffix --units b --options lv_size testvg/snap1"] = "  1073741824"

	params := plugin.Params{
		Type:        "snapshot",
		Source:      "source1",
		Filesystem:  "ext4",
		Mode:        "filesystem",
		VolumeGroup: "testvg",
		ThinPool:    "thinpool",
		MountDir:    mountDir,
	}
	req := testRequest("create", "snap1", 0, params)
	resp, err := p.Create(req)
	must.NoError(t, err)
	must.Eq(t, int64(1073741824), resp.Bytes)
}

func TestCreate_Snapshot_MissingSource(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	params := plugin.Params{
		Type:        "snapshot",
		Mode:        "filesystem",
		VolumeGroup: "testvg",
		ThinPool:    "thinpool",
		MountDir:    t.TempDir(),
	}
	req := testRequest("create", "snap1", 0, params)
	_, err := p.Create(req)
	must.ErrorContains(t, err, "source is required")
}

func TestCreate_Snapshot_SourceNotExists(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	// source doesn't exist
	m.errors["/usr/sbin/lvs --noheadings --nosuffix testvg/source1"] = fmt.Errorf("not found")

	params := plugin.Params{
		Type:        "snapshot",
		Source:      "source1",
		Mode:        "filesystem",
		VolumeGroup: "testvg",
		ThinPool:    "thinpool",
		MountDir:    t.TempDir(),
	}
	req := testRequest("create", "snap1", 0, params)
	_, err := p.Create(req)
	must.ErrorContains(t, err, "does not exist")
}

func TestDelete(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	req := testRequest("delete", "vol1", 0, persistentParams(t.TempDir()))
	err := p.Delete(req)
	must.NoError(t, err)
}

func TestDelete_MissingVolumeID(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	req := testRequest("delete", "", 0, persistentParams(t.TempDir()))
	err := p.Delete(req)
	must.ErrorContains(t, err, "DHV_VOLUME_ID")
}

func TestDelete_InvalidVolumeID(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	req := testRequest("delete", "../escape", 0, persistentParams(t.TempDir()))
	err := p.Delete(req)
	must.Error(t, err)
}

func TestCreate_Snapshot_BlockMode(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	// snapshot doesn't exist
	m.errors["/usr/sbin/lvs --noheadings --nosuffix testvg/snap1"] = fmt.Errorf("not found")
	// SizeBytes
	m.outputs["/usr/sbin/lvs --noheadings --nosuffix --units b --options lv_size testvg/snap1"] = "  1073741824"

	params := plugin.Params{
		Type:        "snapshot",
		Source:      "source1",
		Mode:        "block",
		VolumeGroup: "testvg",
		ThinPool:    "thinpool",
		MountDir:    t.TempDir(),
	}
	req := testRequest("create", "snap1", 0, params)
	resp, err := p.Create(req)
	must.NoError(t, err)
	must.StrContains(t, resp.Path, "/dev/testvg/snap1")

	// No mount in block mode
	for _, cmd := range m.commands {
		must.StrNotContains(t, cmd, "mount")
	}
}

func TestCreate_MissingVolumeID(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	req := testRequest("create", "", 1073741824, persistentParams(t.TempDir()))
	_, err := p.Create(req)
	must.ErrorContains(t, err, "DHV_VOLUME_ID")
}

func TestCreate_InvalidVolumeID(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	req := testRequest("create", "../escape", 1073741824, persistentParams(t.TempDir()))
	_, err := p.Create(req)
	must.Error(t, err)
}

func TestCreate_UnknownType(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	params := persistentParams(t.TempDir())
	params.Type = "magic"
	req := testRequest("create", "vol1", 1073741824, params)
	_, err := p.Create(req)
	must.ErrorContains(t, err, "unknown volume type")
}

package lvm

import (
	"fmt"
	"strings"
	"testing"

	"github.com/shoenig/test/must"
)

func testPlugin(m *mockExec) *Plugin {
	return NewPlugin(NewClient(m, testBinPath))
}

func testRequest(op, volID string, capMin int64, params Params) *Request {
	return &Request{
		Operation:   op,
		VolumeID:    volID,
		CapacityMin: capMin,
		Params:      params,
	}
}

func persistentParams(mountDir string) Params {
	return Params{
		Type:        "persistent",
		Filesystem:  "ext4",
		Mode:        "filesystem",
		VolumeGroup: "testvg",
		ThinPool:    "thinpool",
		MountDir:    mountDir,
	}
}

// --- ParseRequest tests ---

func TestParseRequest_operation(t *testing.T) {
	t.Run("from env", func(t *testing.T) {
		t.Setenv("DHV_OPERATION", "create")
		t.Setenv("DHV_VOLUME_ID", "")
		t.Setenv("DHV_CAPACITY_MIN_BYTES", "")
		t.Setenv("DHV_PARAMETERS", "")
		req, err := ParseRequest()
		must.NoError(t, err)
		must.Eq(t, "create", req.Operation)
	})

	t.Run("env takes precedence over args", func(t *testing.T) {
		t.Setenv("DHV_OPERATION", "delete")
		t.Setenv("DHV_VOLUME_ID", "")
		t.Setenv("DHV_CAPACITY_MIN_BYTES", "")
		t.Setenv("DHV_PARAMETERS", "")
		req, err := ParseRequest()
		must.NoError(t, err)
		must.Eq(t, "delete", req.Operation)
	})
}

func TestParseRequest_volumeID(t *testing.T) {
	t.Setenv("DHV_OPERATION", "create")
	t.Setenv("DHV_VOLUME_ID", "my-vol")
	t.Setenv("DHV_CAPACITY_MIN_BYTES", "1024")
	t.Setenv("DHV_PARAMETERS", "")

	req, err := ParseRequest()
	must.NoError(t, err)
	must.Eq(t, "my-vol", req.VolumeID)
}

func TestParseRequest_capacity(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		t.Setenv("DHV_OPERATION", "create")
		t.Setenv("DHV_VOLUME_ID", "")
		t.Setenv("DHV_CAPACITY_MIN_BYTES", "10485760")
		t.Setenv("DHV_PARAMETERS", "")
		req, err := ParseRequest()
		must.NoError(t, err)
		must.Eq(t, int64(10485760), req.CapacityMin)
	})

	t.Run("empty is zero", func(t *testing.T) {
		t.Setenv("DHV_OPERATION", "create")
		t.Setenv("DHV_VOLUME_ID", "")
		t.Setenv("DHV_CAPACITY_MIN_BYTES", "")
		t.Setenv("DHV_PARAMETERS", "")
		req, err := ParseRequest()
		must.NoError(t, err)
		must.Eq(t, int64(0), req.CapacityMin)
	})

	t.Run("not a number", func(t *testing.T) {
		t.Setenv("DHV_OPERATION", "create")
		t.Setenv("DHV_CAPACITY_MIN_BYTES", "abc")
		t.Setenv("DHV_PARAMETERS", "")
		_, err := ParseRequest()
		must.Error(t, err)
	})

	t.Run("zero", func(t *testing.T) {
		t.Setenv("DHV_OPERATION", "create")
		t.Setenv("DHV_CAPACITY_MIN_BYTES", "0")
		t.Setenv("DHV_PARAMETERS", "")
		req, err := ParseRequest()
		must.NoError(t, err)
		must.Eq(t, int64(0), req.CapacityMin)
	})

	t.Run("negative", func(t *testing.T) {
		t.Setenv("DHV_OPERATION", "create")
		t.Setenv("DHV_CAPACITY_MIN_BYTES", "-100")
		t.Setenv("DHV_PARAMETERS", "")
		req, err := ParseRequest()
		must.NoError(t, err)
		must.Eq(t, int64(-100), req.CapacityMin)
	})
}

func TestParseRequest_parameters(t *testing.T) {
	t.Run("empty defaults", func(t *testing.T) {
		t.Setenv("DHV_OPERATION", "create")
		t.Setenv("DHV_VOLUME_ID", "")
		t.Setenv("DHV_CAPACITY_MIN_BYTES", "")
		t.Setenv("DHV_PARAMETERS", "")
		req, err := ParseRequest()
		must.NoError(t, err)
		must.Eq(t, "persistent", req.Params.Type)
		must.Eq(t, "ext4", req.Params.Filesystem)
		must.Eq(t, "filesystem", req.Params.Mode)
	})

	t.Run("empty object", func(t *testing.T) {
		t.Setenv("DHV_OPERATION", "create")
		t.Setenv("DHV_VOLUME_ID", "")
		t.Setenv("DHV_CAPACITY_MIN_BYTES", "")
		t.Setenv("DHV_PARAMETERS", "{}")
		req, err := ParseRequest()
		must.NoError(t, err)
		must.Eq(t, "persistent", req.Params.Type)
		must.Eq(t, "filesystem", req.Params.Mode)
	})

	t.Run("snapshot with source", func(t *testing.T) {
		t.Setenv("DHV_OPERATION", "create")
		t.Setenv("DHV_VOLUME_ID", "")
		t.Setenv("DHV_CAPACITY_MIN_BYTES", "")
		t.Setenv("DHV_PARAMETERS", `{"type":"snapshot","source":"golden","volume_group":"vg","thin_pool":"pool"}`)
		req, err := ParseRequest()
		must.NoError(t, err)
		must.Eq(t, "snapshot", req.Params.Type)
		must.Eq(t, "golden", req.Params.Source)
		must.Eq(t, "vg", req.Params.VolumeGroup)
		must.Eq(t, "pool", req.Params.ThinPool)
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Setenv("DHV_OPERATION", "create")
		t.Setenv("DHV_CAPACITY_MIN_BYTES", "")
		t.Setenv("DHV_PARAMETERS", "not json")
		_, err := ParseRequest()
		must.Error(t, err)
	})

	t.Run("block mode", func(t *testing.T) {
		t.Setenv("DHV_OPERATION", "create")
		t.Setenv("DHV_VOLUME_ID", "")
		t.Setenv("DHV_CAPACITY_MIN_BYTES", "")
		t.Setenv("DHV_PARAMETERS", `{"mode":"block"}`)
		req, err := ParseRequest()
		must.NoError(t, err)
		must.Eq(t, "block", req.Params.Mode)
	})

	t.Run("invalid mode", func(t *testing.T) {
		t.Setenv("DHV_OPERATION", "create")
		t.Setenv("DHV_VOLUME_ID", "")
		t.Setenv("DHV_CAPACITY_MIN_BYTES", "")
		t.Setenv("DHV_PARAMETERS", `{"mode":"raw"}`)
		_, err := ParseRequest()
		must.ErrorContains(t, err, "invalid mode")
	})

	t.Run("invalid type", func(t *testing.T) {
		t.Setenv("DHV_OPERATION", "create")
		t.Setenv("DHV_VOLUME_ID", "")
		t.Setenv("DHV_CAPACITY_MIN_BYTES", "")
		t.Setenv("DHV_PARAMETERS", `{"type":"magic"}`)
		_, err := ParseRequest()
		must.ErrorContains(t, err, "invalid type")
	})
}

// --- Config tests ---

func TestConfigFromParams(t *testing.T) {
	t.Run("valid minimal with defaults", func(t *testing.T) {
		p := &Params{VolumeGroup: "myvg", ThinPool: "mypool"}
		cfg, err := configFromParams(p)
		must.NoError(t, err)
		must.Eq(t, "myvg", cfg.VolumeGroup)
		must.Eq(t, "mypool", cfg.ThinPool)
		must.Eq(t, "/srv/nomad-volumes", cfg.MountDir)
		must.Eq(t, "/usr/sbin", cfg.BinPath)
	})

	t.Run("all params explicit", func(t *testing.T) {
		p := &Params{
			VolumeGroup: "myvg",
			ThinPool:    "mypool",
			MountDir:    "/custom/mounts",
			BinPath:     "/nix/store/lvm2/bin",
		}
		cfg, err := configFromParams(p)
		must.NoError(t, err)
		must.Eq(t, "/custom/mounts", cfg.MountDir)
		must.Eq(t, "/nix/store/lvm2/bin", cfg.BinPath)
	})

	t.Run("missing volume_group", func(t *testing.T) {
		p := &Params{ThinPool: "mypool"}
		_, err := configFromParams(p)
		must.ErrorContains(t, err, "volume_group is required")
	})

	t.Run("missing thin_pool", func(t *testing.T) {
		p := &Params{VolumeGroup: "myvg"}
		_, err := configFromParams(p)
		must.ErrorContains(t, err, "thin_pool is required")
	})
}

// --- Plugin tests ---

func TestCreate_Persistent(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)
	mountDir := t.TempDir()

	m.errors["/usr/sbin/lvs --noheadings --nosuffix testvg/vol1"] = fmt.Errorf("not found")
	m.outputs["/usr/sbin/lvs --noheadings --nosuffix --units b --options lv_size testvg/vol1"] = "  1073741824"

	req := testRequest("create", "vol1", 1073741824, persistentParams(mountDir))
	resp, err := p.create(req)
	must.NoError(t, err)
	must.Eq(t, int64(1073741824), resp.Bytes)
	must.StrContains(t, resp.Path, "vol1")
}

func TestCreate_Persistent_RollbackOnActivateFail(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	m.errors["/usr/sbin/lvs --noheadings --nosuffix testvg/vol1"] = fmt.Errorf("not found")
	m.errors["/usr/sbin/lvchange --activate y testvg/vol1"] = fmt.Errorf("activate failed")

	req := testRequest("create", "vol1", 1073741824, persistentParams(t.TempDir()))
	_, err := p.create(req)
	must.ErrorContains(t, err, "lvchange activate")
}

func TestCreate_Persistent_Idempotent(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	// lvs succeeds → volume already exists
	m.outputs["/usr/sbin/lvs --noheadings --nosuffix --units b --options lv_size testvg/vol1"] = "  1073741824"

	req := testRequest("create", "vol1", 1073741824, persistentParams(t.TempDir()))
	resp, err := p.create(req)
	must.NoError(t, err)
	must.Eq(t, int64(1073741824), resp.Bytes)

	// Should NOT have called lvcreate (volume existed)
	for _, cmd := range m.commands {
		must.StrNotContains(t, cmd, "lvcreate")
	}
	// But SHOULD have activated (may be inactive after reboot)
	activated := false
	for _, cmd := range m.commands {
		if strings.Contains(cmd, "lvchange") && strings.Contains(cmd, "--activate") {
			activated = true
		}
	}
	must.True(t, activated)
}

func TestCreate_Persistent_BlockMode(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	m.errors["/usr/sbin/lvs --noheadings --nosuffix testvg/vol1"] = fmt.Errorf("not found")
	m.outputs["/usr/sbin/lvs --noheadings --nosuffix --units b --options lv_size testvg/vol1"] = "  1073741824"

	params := persistentParams(t.TempDir())
	params.Mode = "block"
	req := testRequest("create", "vol1", 1073741824, params)
	resp, err := p.create(req)
	must.NoError(t, err)
	must.StrContains(t, resp.Path, "/dev/testvg/vol1")

	for _, cmd := range m.commands {
		must.StrNotContains(t, cmd, "mount")
	}
}

func TestCreate_Snapshot(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)
	mountDir := t.TempDir()

	m.errors["/usr/sbin/lvs --noheadings --nosuffix testvg/snap1"] = fmt.Errorf("not found")
	m.outputs["/usr/sbin/lvs --noheadings --nosuffix --units b --options lv_size testvg/snap1"] = "  1073741824"

	params := Params{
		Type:        "snapshot",
		Source:      "source1",
		Filesystem:  "ext4",
		Mode:        "filesystem",
		VolumeGroup: "testvg",
		ThinPool:    "thinpool",
		MountDir:    mountDir,
	}
	req := testRequest("create", "snap1", 0, params)
	resp, err := p.create(req)
	must.NoError(t, err)
	must.Eq(t, int64(1073741824), resp.Bytes)
}

func TestCreate_Snapshot_MissingSource(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	params := Params{
		Type:        "snapshot",
		Mode:        "filesystem",
		VolumeGroup: "testvg",
		ThinPool:    "thinpool",
		MountDir:    t.TempDir(),
	}
	req := testRequest("create", "snap1", 0, params)
	_, err := p.create(req)
	must.ErrorContains(t, err, "source is required")
}

func TestCreate_Snapshot_SourceNotExists(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	m.errors["/usr/sbin/lvs --noheadings --nosuffix testvg/source1"] = fmt.Errorf("not found")

	params := Params{
		Type:        "snapshot",
		Source:      "source1",
		Mode:        "filesystem",
		VolumeGroup: "testvg",
		ThinPool:    "thinpool",
		MountDir:    t.TempDir(),
	}
	req := testRequest("create", "snap1", 0, params)
	_, err := p.create(req)
	must.ErrorContains(t, err, "does not exist")
}

func TestDelete(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	req := testRequest("delete", "vol1", 0, persistentParams(t.TempDir()))
	err := p.delete(req)
	must.NoError(t, err)
}

func TestDelete_MissingVolumeID(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	req := testRequest("delete", "", 0, persistentParams(t.TempDir()))
	err := p.delete(req)
	must.ErrorContains(t, err, "DHV_VOLUME_ID")
}

func TestDelete_InvalidVolumeID(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	req := testRequest("delete", "../escape", 0, persistentParams(t.TempDir()))
	err := p.delete(req)
	must.Error(t, err)
}

func TestCreate_Snapshot_BlockMode(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	m.errors["/usr/sbin/lvs --noheadings --nosuffix testvg/snap1"] = fmt.Errorf("not found")
	m.outputs["/usr/sbin/lvs --noheadings --nosuffix --units b --options lv_size testvg/snap1"] = "  1073741824"

	params := Params{
		Type:        "snapshot",
		Source:      "source1",
		Mode:        "block",
		VolumeGroup: "testvg",
		ThinPool:    "thinpool",
		MountDir:    t.TempDir(),
	}
	req := testRequest("create", "snap1", 0, params)
	resp, err := p.create(req)
	must.NoError(t, err)
	must.StrContains(t, resp.Path, "/dev/testvg/snap1")

	for _, cmd := range m.commands {
		must.StrNotContains(t, cmd, "mount")
	}
}

func TestCreate_MissingVolumeID(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	req := testRequest("create", "", 1073741824, persistentParams(t.TempDir()))
	_, err := p.create(req)
	must.ErrorContains(t, err, "DHV_VOLUME_ID")
}

func TestCreate_InvalidVolumeID(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	req := testRequest("create", "../escape", 1073741824, persistentParams(t.TempDir()))
	_, err := p.create(req)
	must.Error(t, err)
}

func TestCreate_UnknownType(t *testing.T) {
	m := newMockExec()
	p := testPlugin(m)

	params := persistentParams(t.TempDir())
	params.Type = "magic"
	req := testRequest("create", "vol1", 1073741824, params)
	_, err := p.create(req)
	must.ErrorContains(t, err, "unknown volume type")
}

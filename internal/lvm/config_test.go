package lvm

import (
	"testing"

	"github.com/pigeon-as/nomad-plugin-lvm/plugin"
	"github.com/shoenig/test/must"
)

func TestConfigFromParams(t *testing.T) {
	t.Run("valid minimal with defaults", func(t *testing.T) {
		p := &plugin.Params{VolumeGroup: "myvg", ThinPool: "mypool"}
		cfg, err := ConfigFromParams(p)
		must.NoError(t, err)
		must.Eq(t, "myvg", cfg.VolumeGroup)
		must.Eq(t, "mypool", cfg.ThinPool)
		must.Eq(t, "/srv/nomad-volumes", cfg.MountDir)
		must.Eq(t, "/usr/sbin", cfg.BinPath)
	})

	t.Run("all params explicit", func(t *testing.T) {
		p := &plugin.Params{
			VolumeGroup: "myvg",
			ThinPool:    "mypool",
			MountDir:    "/custom/mounts",
			BinPath:     "/nix/store/lvm2/bin",
		}
		cfg, err := ConfigFromParams(p)
		must.NoError(t, err)
		must.Eq(t, "/custom/mounts", cfg.MountDir)
		must.Eq(t, "/nix/store/lvm2/bin", cfg.BinPath)
	})

	t.Run("missing volume_group", func(t *testing.T) {
		p := &plugin.Params{ThinPool: "mypool"}
		_, err := ConfigFromParams(p)
		must.ErrorContains(t, err, "volume_group is required")
	})

	t.Run("missing thin_pool", func(t *testing.T) {
		p := &plugin.Params{VolumeGroup: "myvg"}
		_, err := ConfigFromParams(p)
		must.ErrorContains(t, err, "thin_pool is required")
	})

	t.Run("both missing", func(t *testing.T) {
		p := &plugin.Params{}
		_, err := ConfigFromParams(p)
		must.ErrorContains(t, err, "volume_group is required")
	})
}

func TestLVPath(t *testing.T) {
	cfg := &Config{VolumeGroup: "testvg"}
	must.Eq(t, "/dev/testvg/vol1", cfg.LVPath("vol1"))
}

func TestMountPath(t *testing.T) {
	cfg := &Config{MountDir: "/opt/nomad-volumes"}
	must.Eq(t, "/opt/nomad-volumes/vol1", cfg.MountPath("vol1"))
}

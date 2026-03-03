package lvm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shoenig/test/must"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name:   "valid",
			config: Config{VolumeGroup: "myvg", ThinPool: "mypool"},
		},
		{
			name:    "missing volume_group",
			config:  Config{ThinPool: "mypool"},
			wantErr: "volume_group is required",
		},
		{
			name:    "missing thin_pool",
			config:  Config{VolumeGroup: "myvg"},
			wantErr: "thin_pool is required",
		},
		{
			name:    "both missing",
			config:  Config{},
			wantErr: "volume_group is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.wantErr != "" {
				must.ErrorContains(t, err, tc.wantErr)
			} else {
				must.NoError(t, err)
			}
		})
	}
}

func TestLVPath(t *testing.T) {
	cfg := &Config{VolumeGroup: "testvg"}
	must.Eq(t, "/dev/testvg/vol1", cfg.LVPath("vol1"))
}

func TestLoadConfig(t *testing.T) {
	t.Run("missing dir", func(t *testing.T) {
		_, err := LoadConfig("/nonexistent/path")
		must.Error(t, err)
	})

	t.Run("valid config", func(t *testing.T) {
		dir := t.TempDir()
		data := []byte(`{"volume_group":"vg","thin_pool":"pool"}`)
		must.NoError(t, os.WriteFile(filepath.Join(dir, configFileName), data, 0644))

		cfg, err := LoadConfig(dir)
		must.NoError(t, err)
		must.Eq(t, "vg", cfg.VolumeGroup)
		must.Eq(t, "pool", cfg.ThinPool)
	})

	t.Run("invalid json", func(t *testing.T) {
		dir := t.TempDir()
		must.NoError(t, os.WriteFile(filepath.Join(dir, configFileName), []byte("not json"), 0644))

		_, err := LoadConfig(dir)
		must.Error(t, err)
	})

	t.Run("invalid config", func(t *testing.T) {
		dir := t.TempDir()
		must.NoError(t, os.WriteFile(filepath.Join(dir, configFileName), []byte(`{}`), 0644))

		_, err := LoadConfig(dir)
		must.ErrorContains(t, err, "volume_group is required")
	})
}

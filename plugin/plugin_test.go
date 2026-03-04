package plugin

import (
	"testing"

	"github.com/shoenig/test/must"
)

func TestParseRequest_operation(t *testing.T) {
	t.Run("from env", func(t *testing.T) {
		t.Setenv(EnvOperation, "create")
		t.Setenv(EnvVolumeID, "")
		t.Setenv(EnvCapacityMin, "")
		t.Setenv(EnvParameters, "")
		req, err := ParseRequest()
		must.NoError(t, err)
		must.Eq(t, "create", req.Operation)
	})

	t.Run("env takes precedence over args", func(t *testing.T) {
		t.Setenv(EnvOperation, "delete")
		t.Setenv(EnvVolumeID, "")
		t.Setenv(EnvCapacityMin, "")
		t.Setenv(EnvParameters, "")
		req, err := ParseRequest()
		must.NoError(t, err)
		must.Eq(t, "delete", req.Operation)
	})
}

func TestParseRequest_volumeID(t *testing.T) {
	t.Setenv(EnvOperation, "create")
	t.Setenv(EnvVolumeID, "my-vol")
	t.Setenv(EnvCapacityMin, "1024")
	t.Setenv(EnvParameters, "")

	req, err := ParseRequest()
	must.NoError(t, err)
	must.Eq(t, "my-vol", req.VolumeID)
}

func TestParseRequest_capacity(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		t.Setenv(EnvOperation, "create")
		t.Setenv(EnvVolumeID, "")
		t.Setenv(EnvCapacityMin, "10485760")
		t.Setenv(EnvParameters, "")
		req, err := ParseRequest()
		must.NoError(t, err)
		must.Eq(t, int64(10485760), req.CapacityMin)
	})

	t.Run("empty is zero", func(t *testing.T) {
		t.Setenv(EnvOperation, "create")
		t.Setenv(EnvVolumeID, "")
		t.Setenv(EnvCapacityMin, "")
		t.Setenv(EnvParameters, "")
		req, err := ParseRequest()
		must.NoError(t, err)
		must.Eq(t, int64(0), req.CapacityMin)
	})

	t.Run("not a number", func(t *testing.T) {
		t.Setenv(EnvOperation, "create")
		t.Setenv(EnvCapacityMin, "abc")
		t.Setenv(EnvParameters, "")
		_, err := ParseRequest()
		must.Error(t, err)
	})

	t.Run("zero", func(t *testing.T) {
		t.Setenv(EnvOperation, "create")
		t.Setenv(EnvCapacityMin, "0")
		t.Setenv(EnvParameters, "")
		req, err := ParseRequest()
		must.NoError(t, err)
		must.Eq(t, int64(0), req.CapacityMin)
	})

	t.Run("negative", func(t *testing.T) {
		t.Setenv(EnvOperation, "create")
		t.Setenv(EnvCapacityMin, "-100")
		t.Setenv(EnvParameters, "")
		req, err := ParseRequest()
		must.NoError(t, err)
		must.Eq(t, int64(-100), req.CapacityMin)
	})
}

func TestParseRequest_parameters(t *testing.T) {
	t.Run("empty defaults", func(t *testing.T) {
		t.Setenv(EnvOperation, "create")
		t.Setenv(EnvVolumeID, "")
		t.Setenv(EnvCapacityMin, "")
		t.Setenv(EnvParameters, "")
		req, err := ParseRequest()
		must.NoError(t, err)
		must.Eq(t, "persistent", req.Parameters.Type)
		must.Eq(t, "ext4", req.Parameters.Filesystem)
		must.Eq(t, "filesystem", req.Parameters.Mode)
	})

	t.Run("empty object", func(t *testing.T) {
		t.Setenv(EnvOperation, "create")
		t.Setenv(EnvVolumeID, "")
		t.Setenv(EnvCapacityMin, "")
		t.Setenv(EnvParameters, "{}")
		req, err := ParseRequest()
		must.NoError(t, err)
		must.Eq(t, "persistent", req.Parameters.Type)
		must.Eq(t, "filesystem", req.Parameters.Mode)
	})

	t.Run("snapshot with source", func(t *testing.T) {
		t.Setenv(EnvOperation, "create")
		t.Setenv(EnvVolumeID, "")
		t.Setenv(EnvCapacityMin, "")
		t.Setenv(EnvParameters, `{"type":"snapshot","source":"golden","volume_group":"vg","thin_pool":"pool"}`)
		req, err := ParseRequest()
		must.NoError(t, err)
		must.Eq(t, "snapshot", req.Parameters.Type)
		must.Eq(t, "golden", req.Parameters.Source)
		must.Eq(t, "vg", req.Parameters.VolumeGroup)
		must.Eq(t, "pool", req.Parameters.ThinPool)
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Setenv(EnvOperation, "create")
		t.Setenv(EnvCapacityMin, "")
		t.Setenv(EnvParameters, "not json")
		_, err := ParseRequest()
		must.Error(t, err)
	})

	t.Run("block mode", func(t *testing.T) {
		t.Setenv(EnvOperation, "create")
		t.Setenv(EnvVolumeID, "")
		t.Setenv(EnvCapacityMin, "")
		t.Setenv(EnvParameters, `{"mode":"block"}`)
		req, err := ParseRequest()
		must.NoError(t, err)
		must.Eq(t, "block", req.Parameters.Mode)
	})

	t.Run("invalid mode", func(t *testing.T) {
		t.Setenv(EnvOperation, "create")
		t.Setenv(EnvVolumeID, "")
		t.Setenv(EnvCapacityMin, "")
		t.Setenv(EnvParameters, `{"mode":"raw"}`)
		_, err := ParseRequest()
		must.ErrorContains(t, err, "invalid mode")
	})
}

func TestEnvConstants(t *testing.T) {
	// Verify our constants match the Nomad-defined names.
	must.Eq(t, "DHV_OPERATION", EnvOperation)
	must.Eq(t, "DHV_PLUGIN_DIR", EnvPluginDir)
	must.Eq(t, "DHV_VOLUME_ID", EnvVolumeID)
	must.Eq(t, "DHV_CAPACITY_MIN_BYTES", EnvCapacityMin)
	must.Eq(t, "DHV_PARAMETERS", EnvParameters)
	must.Eq(t, "DHV_CREATED_PATH", EnvCreatedPath)
}

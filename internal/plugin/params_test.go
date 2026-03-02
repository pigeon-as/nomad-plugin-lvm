package plugin

import (
	"testing"

	"github.com/shoenig/test/must"
)

func TestParseParams(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		t.Setenv("DHV_PARAMETERS", "")
		p, err := ParseParams()
		must.NoError(t, err)
		must.Eq(t, "persistent", p.Type)
		must.Eq(t, "ext4", p.Filesystem)
	})

	t.Run("empty object", func(t *testing.T) {
		t.Setenv("DHV_PARAMETERS", "{}")
		p, err := ParseParams()
		must.NoError(t, err)
		must.Eq(t, "persistent", p.Type)
	})

	t.Run("persistent explicit", func(t *testing.T) {
		t.Setenv("DHV_PARAMETERS", `{"type":"persistent","filesystem":"ext4"}`)
		p, err := ParseParams()
		must.NoError(t, err)
		must.Eq(t, "persistent", p.Type)
		must.Eq(t, "ext4", p.Filesystem)
	})

	t.Run("snapshot", func(t *testing.T) {
		t.Setenv("DHV_PARAMETERS", `{"type":"snapshot","source":"golden"}`)
		p, err := ParseParams()
		must.NoError(t, err)
		must.Eq(t, "snapshot", p.Type)
		must.Eq(t, "golden", p.Source)
	})

	t.Run("defaults filled", func(t *testing.T) {
		t.Setenv("DHV_PARAMETERS", `{"source":"golden"}`)
		p, err := ParseParams()
		must.NoError(t, err)
		must.Eq(t, "persistent", p.Type)
		must.Eq(t, "ext4", p.Filesystem)
		must.Eq(t, "golden", p.Source)
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Setenv("DHV_PARAMETERS", "not json")
		_, err := ParseParams()
		must.Error(t, err)
	})
}

func TestParseCapacity(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		t.Setenv("DHV_CAPACITY_MIN_BYTES", "10485760")
		n, err := ParseCapacity()
		must.NoError(t, err)
		must.Eq(t, int64(10485760), n)
	})

	t.Run("empty", func(t *testing.T) {
		t.Setenv("DHV_CAPACITY_MIN_BYTES", "")
		_, err := ParseCapacity()
		must.Error(t, err)
	})

	t.Run("not a number", func(t *testing.T) {
		t.Setenv("DHV_CAPACITY_MIN_BYTES", "abc")
		_, err := ParseCapacity()
		must.Error(t, err)
	})

	t.Run("zero", func(t *testing.T) {
		t.Setenv("DHV_CAPACITY_MIN_BYTES", "0")
		_, err := ParseCapacity()
		must.ErrorContains(t, err, "must be > 0")
	})

	t.Run("negative", func(t *testing.T) {
		t.Setenv("DHV_CAPACITY_MIN_BYTES", "-100")
		_, err := ParseCapacity()
		must.ErrorContains(t, err, "must be > 0")
	})
}

func TestRequiredEnv(t *testing.T) {
	t.Run("set", func(t *testing.T) {
		t.Setenv("TEST_KEY", "value")
		v, err := RequiredEnv("TEST_KEY")
		must.NoError(t, err)
		must.Eq(t, "value", v)
	})

	t.Run("not set", func(t *testing.T) {
		t.Setenv("TEST_KEY", "")
		_, err := RequiredEnv("TEST_KEY")
		must.ErrorContains(t, err, "TEST_KEY")
	})
}

func TestOperation(t *testing.T) {
	t.Run("from env", func(t *testing.T) {
		t.Setenv("DHV_OPERATION", "create")
		op, err := Operation()
		must.NoError(t, err)
		must.Eq(t, "create", op)
	})

	t.Run("env takes precedence", func(t *testing.T) {
		t.Setenv("DHV_OPERATION", "delete")
		// os.Args would give something else, but env wins.
		op, err := Operation()
		must.NoError(t, err)
		must.Eq(t, "delete", op)
	})
}

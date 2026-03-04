package lvm

import (
	"fmt"
	"testing"

	"github.com/shoenig/test/must"
)

const testBinPath = "/usr/sbin"

// mockExec records commands and returns pre-configured results.
type mockExec struct {
	commands []string
	outputs  map[string]string
	errors   map[string]error
}

func newMockExec() *mockExec {
	return &mockExec{
		outputs: make(map[string]string),
		errors:  make(map[string]error),
	}
}

func (m *mockExec) key(name string, args ...string) string {
	k := name
	for _, a := range args {
		k += " " + a
	}
	return k
}

func (m *mockExec) Run(name string, args ...string) error {
	_, err := m.Output(name, args...)
	return err
}

func (m *mockExec) Output(name string, args ...string) (string, error) {
	k := m.key(name, args...)
	m.commands = append(m.commands, k)
	if err, ok := m.errors[k]; ok {
		return "", err
	}
	if out, ok := m.outputs[k]; ok {
		return out, nil
	}
	return "", nil
}

func (m *mockExec) whenOutput(output string, name string, args ...string) {
	m.outputs[m.key(name, args...)] = output
}

func (m *mockExec) whenError(err error, name string, args ...string) {
	m.errors[m.key(name, args...)] = err
}

func TestExists(t *testing.T) {
	m := newMockExec()
	c := NewClient(m, testBinPath)

	t.Run("exists", func(t *testing.T) {
		must.True(t, c.Exists("myvg", "mylv"))
	})

	t.Run("not exists", func(t *testing.T) {
		m.whenError(fmt.Errorf("not found"), "/usr/sbin/lvs", "--noheadings", "--nosuffix", "myvg/missing")
		must.False(t, c.Exists("myvg", "missing"))
	})
}

func TestRemove(t *testing.T) {
	m := newMockExec()
	c := NewClient(m, testBinPath)

	t.Run("exists and removed", func(t *testing.T) {
		err := c.Remove("myvg", "vol1")
		must.NoError(t, err)
	})

	t.Run("does not exist is noop", func(t *testing.T) {
		m.whenError(fmt.Errorf("not found"), "/usr/sbin/lvs", "--noheadings", "--nosuffix", "myvg/gone")
		err := c.Remove("myvg", "gone")
		must.NoError(t, err)
	})
}

func TestSizeBytes(t *testing.T) {
	m := newMockExec()
	c := NewClient(m, testBinPath)

	m.whenOutput("  10485760\n", "/usr/sbin/lvs",
		"--noheadings", "--nosuffix", "--units", "b",
		"--options", "lv_size",
		"myvg/vol1")

	size, err := c.SizeBytes("myvg", "vol1")
	must.NoError(t, err)
	must.Eq(t, int64(10485760), size)
}

func TestSizeBytes_error(t *testing.T) {
	m := newMockExec()
	c := NewClient(m, testBinPath)

	m.whenError(fmt.Errorf("lvs failed"), "/usr/sbin/lvs",
		"--noheadings", "--nosuffix", "--units", "b",
		"--options", "lv_size",
		"myvg/missing")

	_, err := c.SizeBytes("myvg", "missing")
	must.Error(t, err)
}

func TestMakeFilesystem_unsupported(t *testing.T) {
	m := newMockExec()
	c := NewClient(m, testBinPath)

	err := c.MakeFilesystem("xfs", "/dev/myvg/vol1")
	must.ErrorContains(t, err, "unsupported filesystem type")
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"simple", "myvolume", false},
		{"with dots", "my.volume", false},
		{"with dashes", "my-volume", false},
		{"with underscores", "my_volume", false},
		{"leading dot", ".hidden", true},
		{"leading underscore", "_internal", false},
		{"numeric", "123", false},
		{"empty", "", true},
		{"has slash", "my/volume", true},
		{"has space", "my volume", true},
		{"starts with dash", "-invalid", true},
		{"special chars", "vol@ume!", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateName(tc.input)
			if tc.wantErr {
				must.Error(t, err)
			} else {
				must.NoError(t, err)
			}
		})
	}
}

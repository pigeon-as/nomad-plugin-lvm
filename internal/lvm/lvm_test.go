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
	c := New(m, testBinPath)

	t.Run("exists", func(t *testing.T) {
		must.True(t, c.Exists("myvg", "mylv"))
	})

	t.Run("not exists", func(t *testing.T) {
		m.whenError(fmt.Errorf("not found"), "/usr/sbin/lvs", "--noheadings", "--nosuffix", "myvg/missing")
		must.False(t, c.Exists("myvg", "missing"))
	})
}

func TestCreateThin(t *testing.T) {
	m := newMockExec()
	c := New(m, testBinPath)

	err := c.CreateThin("myvg", "mypool", "vol1", 1048576)
	must.NoError(t, err)

	must.SliceContains(t, m.commands,
		"/usr/sbin/lvcreate --thin --virtualsize 1048576b --thinpool mypool --name vol1 myvg")
}

func TestCreateSnapshot(t *testing.T) {
	m := newMockExec()
	c := New(m, testBinPath)

	err := c.CreateSnapshot("myvg", "source", "snap1")
	must.NoError(t, err)

	must.SliceContains(t, m.commands,
		"/usr/sbin/lvcreate --snapshot --name snap1 --setactivationskip n myvg/source")
}

func TestRemove(t *testing.T) {
	m := newMockExec()
	c := New(m, testBinPath)

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

func TestActivate(t *testing.T) {
	m := newMockExec()
	c := New(m, testBinPath)

	err := c.Activate("myvg", "vol1")
	must.NoError(t, err)

	must.SliceContains(t, m.commands,
		"/usr/sbin/lvchange --activate y myvg/vol1")
}

func TestSizeBytes(t *testing.T) {
	m := newMockExec()
	c := New(m, testBinPath)

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
	c := New(m, testBinPath)

	m.whenError(fmt.Errorf("lvs failed"), "/usr/sbin/lvs",
		"--noheadings", "--nosuffix", "--units", "b",
		"--options", "lv_size",
		"myvg/missing")

	_, err := c.SizeBytes("myvg", "missing")
	must.Error(t, err)
}

func TestMakeFilesystem(t *testing.T) {
	m := newMockExec()
	c := New(m, testBinPath)

	t.Run("ext4", func(t *testing.T) {
		err := c.MakeFilesystem("ext4", "/dev/myvg/vol1")
		must.NoError(t, err)
		must.SliceContains(t, m.commands, "/usr/sbin/mkfs.ext4 -q /dev/myvg/vol1")
	})

	t.Run("unsupported", func(t *testing.T) {
		err := c.MakeFilesystem("xfs", "/dev/myvg/vol1")
		must.ErrorContains(t, err, "unsupported filesystem type")
	})
}

func TestMount(t *testing.T) {
	m := newMockExec()
	c := New(m, testBinPath)

	err := c.Mount("/dev/myvg/vol1", "/mnt/vol1")
	must.NoError(t, err)
	must.SliceContains(t, m.commands, "/usr/sbin/mount /dev/myvg/vol1 /mnt/vol1")
}

func TestUnmount(t *testing.T) {
	m := newMockExec()
	c := New(m, testBinPath)

	err := c.Unmount("/mnt/vol1")
	must.NoError(t, err)
	must.SliceContains(t, m.commands, "/usr/sbin/umount /mnt/vol1")
}

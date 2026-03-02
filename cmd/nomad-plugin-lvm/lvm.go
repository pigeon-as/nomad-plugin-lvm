package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func lvExists(vg, lv string) bool {
	err := run("lvs", "--noheadings", "--nosuffix", fmt.Sprintf("%s/%s", vg, lv))
	return err == nil
}

func lvCreateThin(vg, thinPool, name string, sizeBytes int64) error {
	size := fmt.Sprintf("%db", sizeBytes)
	return run("lvcreate",
		"--thin",
		"--virtualsize", size,
		"--thinpool", thinPool,
		"--name", name,
		vg,
	)
}

func lvCreateSnapshot(vg, source, name string) error {
	origin := fmt.Sprintf("%s/%s", vg, source)
	return run("lvcreate",
		"--snapshot",
		"--name", name,
		"--setactivationskip", "n",
		origin,
	)
}

func lvRemove(vg, name string) error {
	if !lvExists(vg, name) {
		return nil
	}
	return run("lvremove", "--force", fmt.Sprintf("%s/%s", vg, name))
}

func lvActivate(vg, name string) error {
	return run("lvchange", "--activate", "y", fmt.Sprintf("%s/%s", vg, name))
}

func lvSizeBytes(vg, name string) (int64, error) {
	out, err := output("lvs",
		"--noheadings", "--nosuffix", "--units", "b",
		"--options", "lv_size",
		fmt.Sprintf("%s/%s", vg, name),
	)
	if err != nil {
		return 0, err
	}
	var size int64
	if _, err := fmt.Sscan(strings.TrimSpace(out), &size); err != nil {
		return 0, fmt.Errorf("parsing lv_size %q: %w", out, err)
	}
	return size, nil
}

func mkfs(fsType, device string) error {
	if fsType != "ext4" {
		return fmt.Errorf("unsupported filesystem type: %q", fsType)
	}
	return run("mkfs.ext4", "-q", device)
}

func run(name string, args ...string) error {
	_, err := output(name, args...)
	return err
}

func output(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

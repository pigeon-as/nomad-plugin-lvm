package lvm

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Exec runs system commands.
type Exec interface {
	Run(name string, args ...string) error
	Output(name string, args ...string) (string, error)
}

// ExecCommand executes commands via os/exec.
type ExecCommand struct{}

func (e ExecCommand) Run(name string, args ...string) error {
	_, err := e.Output(name, args...)
	return err
}

func (ExecCommand) Output(name string, args ...string) (string, error) {
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

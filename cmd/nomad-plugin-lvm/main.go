package main

import (
	"fmt"
	"os"

	"github.com/pigeon-as/nomad-plugin-lvm/internal/lvm"
	"github.com/pigeon-as/nomad-plugin-lvm/internal/plugin"
)

func main() {
	op, err := plugin.Operation()
	if err != nil {
		fatalf("%v", err)
	}

	// Fingerprint does not require config or LVM.
	if op == "fingerprint" {
		p := &plugin.Plugin{Stdout: os.Stdout}
		if err := p.Fingerprint(); err != nil {
			fatalf("fingerprint: %v", err)
		}
		return
	}

	cfg, err := plugin.LoadConfig(os.Getenv("DHV_PLUGIN_DIR"))
	if err != nil {
		plugin.WriteError(os.Stdout, fmt.Errorf("config: %w", err))
		os.Exit(1)
	}

	p := plugin.New(cfg, lvm.New(lvm.SystemRunner{}))

	switch op {
	case "create":
		err = p.Create()
	case "delete":
		err = p.Delete()
	default:
		fatalf("unknown operation: %s", op)
	}
	if err != nil {
		plugin.WriteError(os.Stdout, err)
		os.Exit(1)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

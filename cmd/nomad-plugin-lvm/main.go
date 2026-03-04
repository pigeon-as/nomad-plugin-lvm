package main

import (
	"fmt"
	"os"

	"github.com/pigeon-as/nomad-plugin-lvm/internal/lvm"
	"github.com/pigeon-as/nomad-plugin-lvm/plugin"
)

func main() {
	req, err := plugin.ParseRequest()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// For fingerprint, config may not be available yet — we only
	// need the LVM client for create/delete operations.
	var cfg *lvm.Config
	if req.Operation != "fingerprint" {
		cfg, err = lvm.ConfigFromParams(&req.Parameters)
		if err != nil {
			fmt.Fprintf(os.Stderr, "config: %v\n", err)
			os.Exit(1)
		}
	}

	binPath := "/usr/sbin"
	if cfg != nil {
		binPath = cfg.BinPath
	}
	p := lvm.NewPlugin(lvm.New(lvm.ExecCommand{}, binPath))

	if err := plugin.Run(p, req); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

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

	cfg, err := lvm.LoadConfig(os.Getenv(plugin.EnvPluginDir))
	if req.Operation != "fingerprint" && err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	if err := plugin.Run(lvm.NewPlugin(cfg, lvm.New(lvm.ExecCommand{})), req); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

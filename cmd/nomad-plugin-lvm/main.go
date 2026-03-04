package main

import (
	"fmt"
	"os"

	"github.com/pigeon-as/nomad-plugin-lvm/internal/lvm"
)

func main() {
	// Nomad DHV plugins are launched with a stripped environment (only DHV_*
	// variables). Ensure PATH is set so exec.LookPath can find system binaries.
	if os.Getenv("PATH") == "" {
		os.Setenv("PATH", "/usr/sbin:/usr/bin:/sbin:/bin")
	}

	req, err := lvm.ParseRequest()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	client, err := lvm.NewClient(lvm.ExecCommand{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	p := lvm.NewPlugin(client)
	if err := p.Run(req); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

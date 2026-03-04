package main

import (
	"fmt"
	"os"

	"github.com/pigeon-as/nomad-plugin-lvm/internal/lvm"
)

func main() {
	req, err := lvm.ParseRequest()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	binPath := req.Params.BinPath
	if binPath == "" {
		binPath = lvm.DefaultBinPath
	}

	p := lvm.NewPlugin(lvm.NewClient(lvm.ExecCommand{}, binPath))
	if err := p.Run(req); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

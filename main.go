package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fatalf("usage: nomad-plugin-lvm <fingerprint|create|delete>")
	}

	if os.Args[1] == "fingerprint" {
		if err := cmdFingerprint(); err != nil {
			fatalf("fingerprint: %v", err)
		}
		return
	}

	cfg, err := loadConfig()
	if err != nil {
		writeError(fmt.Errorf("config: %w", err))
		os.Exit(1)
	}

	switch os.Args[1] {
	case "create":
		err = cmdCreate(cfg)
	case "delete":
		err = cmdDelete(cfg)
	default:
		fatalf("unknown operation: %s", os.Args[1])
	}
	if err != nil {
		writeError(err)
		os.Exit(1)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func writeError(err error) {
	if encErr := json.NewEncoder(os.Stdout).Encode(map[string]string{"error": err.Error()}); encErr != nil {
		fmt.Fprintf(os.Stderr, "failed to write JSON error response: %v\n", encErr)
	}
}

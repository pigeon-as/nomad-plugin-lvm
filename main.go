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

	cfg, err := loadConfig()
	if err != nil {
		fatalf("config: %v", err)
	}

	switch os.Args[1] {
	case "fingerprint":
		if err := cmdFingerprint(cfg); err != nil {
			fatalf("fingerprint: %v", err)
		}
	case "create":
		if err := cmdCreate(cfg); err != nil {
			writeError(err)
			os.Exit(1)
		}
	case "delete":
		if err := cmdDelete(cfg); err != nil {
			writeError(err)
			os.Exit(1)
		}
	default:
		fatalf("unknown operation: %s", os.Args[1])
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

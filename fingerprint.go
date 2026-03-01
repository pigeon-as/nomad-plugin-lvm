package main

import (
	"encoding/json"
	"os"
)

func cmdFingerprint(cfg *Config) error {
	return json.NewEncoder(os.Stdout).Encode(map[string]string{
		"version": pluginVersion,
	})
}

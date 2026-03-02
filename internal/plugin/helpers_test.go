package plugin

import (
	"os"
	"path/filepath"
)

// writeFile is a test helper to create a file in a directory.
func writeFile(dir, name string, data []byte) error {
	return os.WriteFile(filepath.Join(dir, name), data, 0644)
}

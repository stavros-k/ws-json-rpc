package utils

import (
	"os"
)

// FileExists checks if a file exists.
// Returns true if the file exists at the given path, false otherwise.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

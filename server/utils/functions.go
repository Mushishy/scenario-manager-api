package utils

import "os"

// ensureDirectoryExists creates the directory if it doesn't exist
func EnsureDirectoryExists(path string) {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		panic("Failed to create directory: " + path)
	}
}

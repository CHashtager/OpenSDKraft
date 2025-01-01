package utils

import (
	"os"
)

// CreateDirectory creates a directory if it doesn't exist
func CreateDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

// WriteFile writes content to a file, creating it if it doesn't exist
func WriteFile(filename string, content []byte) error {
	return os.WriteFile(filename, content, 0644)
}

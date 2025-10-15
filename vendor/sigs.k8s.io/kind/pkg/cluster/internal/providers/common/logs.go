package common

import (
	"os"
	"path/filepath"
)

// FileOnHost is a helper to create a file at path
// even if the parent directory doesn't exist
// in which case it will be created with ModePerm
func FileOnHost(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return nil, err
	}
	return os.Create(path)
}

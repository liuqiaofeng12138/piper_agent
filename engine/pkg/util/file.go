package util

import (
	"os"
	"path/filepath"
)

func ReadFileLines(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func EnsureDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

func WriteFile(path string, data []byte, perm os.FileMode) error {
	if err := EnsureDir(path); err != nil {
		return err
	}
	return os.WriteFile(path, data, perm)
}

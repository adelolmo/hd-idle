package io

import (
	"os"
	"path/filepath"
)

func RealPath(path string) string {
	if path[0] != '/' {
		return path
	}

	s, err := os.Readlink(path)
	if err == nil {
		return filepath.Base(s)
	}

	return ""
}

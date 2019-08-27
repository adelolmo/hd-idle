package io

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func RealPath(path string) (string, error) {
	if path[0] != '/' {
		return path, nil
	}
	if !strings.Contains(path, "by-") {
		return filepath.Base(path), nil
	}
	s, err := os.Readlink(path)
	if err == nil {
		return filepath.Base(s), nil
	}

	return "", fmt.Errorf("cannot find device for %s", path)
}

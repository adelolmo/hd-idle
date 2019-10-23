package io

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
		device := filepath.Base(s)
		/* remove partition numbers, if any */
		for {
			i := device[len(device)-1:]
			_, err := strconv.Atoi(i)
			if err != nil {
				break
			}
			device = device[:len(device)-1]
		}
		return device, nil
	}

	return "", fmt.Errorf("cannot find device for %s", path)
}

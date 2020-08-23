// hd-idle - spin down idle hard disks
// Copyright (C) 2018  Andoni del Olmo
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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

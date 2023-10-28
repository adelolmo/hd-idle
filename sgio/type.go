// hd-idle - spin down idle hard disks
// Copyright (C) 2023  Andoni del Olmo
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

package sgio

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	Jmicron = iota
	Unknown = iota

	sysblock = "/sys/block"

	jmicron = 0x152d
)

type apt struct {
	idVendor, idProduct, bcdDevice int
}

func (a apt) isJmicron() bool {
	if a.idVendor != jmicron {
		return false
	}
	switch a.idProduct {
	case 0x2329, 0x2336, 0x2338, 0x2339:
		return true
	}
	return false
}

func (ad ataDevice) deviceType() int {
	a, err := ad.identifyDevice(ad.device)
	if err != nil {
		return Unknown
	}
	if a.isJmicron() {
		if ad.debug {
			fmt.Println("APT: Found supported device jmicron")
		}
		return Jmicron
	}

	return Unknown
}

func (ad ataDevice) identifyDevice(device string) (apt, error) {
	diskname := strings.Split(device, "/")[2]
	sysblockdisk := filepath.Join(sysblock, diskname)
	idVendor, err := findSystemFile(sysblockdisk, "idVendor")
	if err != nil {
		return apt{}, err
	}
	idProduct, err := findSystemFile(sysblockdisk, "idProduct")
	if err != nil {
		return apt{}, err
	}
	bcdDevice, err := findSystemFile(sysblockdisk, "bcdDevice")
	if err != nil {
		return apt{}, err
	}
	if ad.debug {
		fmt.Printf("APT: USB ID = 0x%d:0x%d (0x%3d)\n", idVendor, idProduct, bcdDevice)
	}
	return apt{
			idVendor:  idVendor,
			idProduct: idProduct,
			bcdDevice: bcdDevice,
		},
		nil
}

func findSystemFile(systemRoot, filename string) (int, error) {
	_, err := os.ReadFile(filepath.Join(systemRoot, filename))
	relativeDir := ""
	var content []byte

	depth := 0
	for depth < 20 {
		if err == nil {
			id, err := strconv.Atoi(strings.TrimSpace(string(content)))
			if err != nil {
				return -1, err
			}
			return id, nil
		}
		relativeDir += "/.."
		content, err = os.ReadFile(systemRoot + relativeDir + "/" + filename)
		depth++
	}

	return -1, fmt.Errorf("device not found")
}

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
	"strings"
)

const (
	Jmicron = iota
	Unknown = iota

	sysblock = "/sys/block"

	jmicron = "152d"
)

type AtaDevice struct {
	device string
	debug  bool
	fsRoot string
}

func NewAtaDevice(device string, debug bool) AtaDevice {
	return AtaDevice{
		device: device,
		debug:  debug,
		fsRoot: sysblock,
	}
}

type apt struct {
	idVendor, idProduct, bcdDevice string
}

func (a apt) isJmicron() bool {
	if a.idVendor != jmicron {
		return false
	}
	switch a.idProduct {
	case "2329", "2336", "2338", "2339":
		return true
	}
	return false
}

func (ad AtaDevice) deviceType() int {
	a, err := ad.identifyDevice(ad.device)
	if err != nil {
		if ad.debug {
			fmt.Println("APT: Unsupported device")
		}
		return Unknown
	}
	if a.isJmicron() {
		if ad.debug {
			fmt.Println("APT: Found supported device jmicron")
		}
		return Jmicron
	}

	if ad.debug {
		fmt.Println("APT: Unsupported device")
	}
	return Unknown
}

func (ad AtaDevice) identifyDevice(device string) (apt, error) {
	diskname := strings.Split(device, "/")[2]
	sysblockdisk := filepath.Join(ad.fsRoot, diskname)
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
		fmt.Printf("APT: USB ID = 0x%s:0x%s (0x%3s)\n", idVendor, idProduct, bcdDevice)
	}
	return apt{
			idVendor:  idVendor,
			idProduct: idProduct,
			bcdDevice: bcdDevice,
		},
		nil
}

func findSystemFile(systemRoot, filename string) (string, error) {
	_, err := os.ReadFile(filepath.Join(systemRoot, filename))
	relativeDir := ""
	var content []byte

	depth := 0
	for depth < 20 {
		if err == nil {
			return strings.TrimSpace(string(content)), nil
		}
		relativeDir += "/.."
		content, err = os.ReadFile(systemRoot + relativeDir + "/" + filename)
		depth++
	}

	return "", fmt.Errorf("device not found")
}

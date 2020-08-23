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

package sgio

import (
	"fmt"
	"github.com/benmcclelland/sgio"
	"os"
	"syscall"
	"unsafe"
)

const SgDxferNone = -1

func openDevice(fname string) (*os.File, error) {
	f, err := os.OpenFile(fname, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	var version uint32
	if (ioctl(f.Fd(), sgio.SG_GET_VERSION_NUM, uintptr(unsafe.Pointer(&version))) != nil) || (version < 30000) {
		return nil, fmt.Errorf("device does not appear to be an sg device")
	}
	return f, nil
}

func ioctl(fd, cmd, ptr uintptr) error {
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, cmd, ptr)
	if err != 0 {
		return err
	}
	return nil
}

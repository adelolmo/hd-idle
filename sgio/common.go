package sgio

import (
	"fmt"
	"github.com/benmcclelland/sgio"
	"os"
	"syscall"
	"unsafe"
)

const (
	SG_DXFER_NONE = -1
)

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

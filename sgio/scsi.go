package sgio

import (
	"fmt"
	"github.com/benmcclelland/sgio"
)

// https://en.wikipedia.org/wiki/SCSI_command
const startStopUnit = 0x1b

func StopScsiDevice(device string) error {
	f, err := openDevice(device)
	if err != nil {
		return err
	}

	senseBuf := make([]byte, sgio.SENSE_BUF_LEN)
	inqCmdBlk := []uint8{startStopUnit, 0, 0, 0, 0, 0}
	ioHdr := &sgio.SgIoHdr{
		InterfaceID:    'S',
		DxferDirection: SgDxferNone,
		Cmdp:           &inqCmdBlk[0],
		CmdLen:         uint8(len(inqCmdBlk)),
		Sbp:            &senseBuf[0],
		MxSbLen:        sgio.SENSE_BUF_LEN,
	}

	if err := sgio.SgioSyscall(f, ioHdr); err != nil {
		return err
	}

	if err := sgio.CheckSense(ioHdr, &senseBuf); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("cannot close file %s. Error: %s", device, err)
	}
}

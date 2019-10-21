package sgio

import (
	"fmt"
	"github.com/benmcclelland/sgio"
	"os"
)

const (
	sgAta16    = 0x85
	sgAta16Len = 16

	sgAtaProtoNonData = 3 << 1
	sgCdb2CheckCond   = 1 << 5
	ataUsingLba       = 1 << 6

	ataOpStandbyNow1 = 0xe0 // https://wiki.osdev.org/ATA/ATAPI_Power_Management
	ataOpStandbyNow2 = 0x94 // Retired in ATA4. Did not coexist with ATAPI.
)

func StopAtaDevice(device string) error {
	f, err := openDevice(device)
	if err != nil {
		return err
	}

	if err = sendAtaCommand(f, ataOpStandbyNow1); err != nil {
		return err
	}
	if err = sendAtaCommand(f, ataOpStandbyNow2); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("cannot close file %s. Error: %s", device, err)
	}
	return nil
}

func sendAtaCommand(f *os.File, command uint8) error {
	var cbd [sgAta16Len]uint8
	cbd[0] = sgAta16
	cbd[1] = sgAtaProtoNonData
	cbd[2] = sgCdb2CheckCond
	cbd[13] = ataUsingLba
	cbd[14] = command
	return sendSgio(f, cbd)
}

func sendSgio(f *os.File, inqCmdBlk [sgAta16Len]uint8) error {
	senseBuf := make([]byte, sgio.SENSE_BUF_LEN)
	ioHdr := &sgio.SgIoHdr{
		InterfaceID:    'S',
		DxferDirection: SgDxferNone,
		Cmdp:           &inqCmdBlk[0],
		CmdLen:         sgAta16Len,
		Sbp:            &senseBuf[0],
		MxSbLen:        sgio.SENSE_BUF_LEN,
		Timeout:        0,
	}

	if err := sgio.SgioSyscall(f, ioHdr); err != nil {
		return err
	}

	if err := sgio.CheckSense(ioHdr, &senseBuf); err != nil {
		return err
	}
	return nil
}

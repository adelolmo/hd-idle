package sgio

import (
	"github.com/benmcclelland/sgio"
	"log"
)

// https://en.wikipedia.org/wiki/SCSI_command
const startStopUnit = 0x1b

func StopScsiDevice(device string) {
	f, err := openDevice(device)
	if err != nil {
		log.Fatalln(err)
	}

	senseBuf := make([]byte, sgio.SENSE_BUF_LEN)
	inqCmdBlk := []uint8{startStopUnit, 0, 0, 0, 0, 0}
	ioHdr := &sgio.SgIoHdr{
		InterfaceID:    int32('S'),
		DxferDirection: SgDxferNone,
		Cmdp:           &inqCmdBlk[0],
		CmdLen:         uint8(len(inqCmdBlk)),
		Sbp:            &senseBuf[0],
		MxSbLen:        sgio.SENSE_BUF_LEN,
	}

	if err := sgio.SgioSyscall(f, ioHdr); err != nil {
		log.Fatalln(err)
	}

	if err := sgio.CheckSense(ioHdr, &senseBuf); err != nil {
		log.Fatalln(err)
	}

	if err := f.Close(); err != nil {
		log.Fatalf("Cannot close file %s. Error: %s", device, err)
	}
}

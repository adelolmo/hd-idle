package sgio

import (
	"github.com/benmcclelland/sgio"
	"log"
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

func StopAtaDevice(device string) {
	f, err := openDevice(device)
	if err != nil {
		log.Fatalln(err)
	}

	sendAtaCommand(f, ataOpStandbyNow1)
	sendAtaCommand(f, ataOpStandbyNow2)

	if err := f.Close(); err != nil {
		log.Fatalf("Cannot close file %s. Error: %s", device, err)
	}
}

func sendAtaCommand(f *os.File, command uint8) {
	var cbd [sgAta16Len]uint8
	cbd[0] = sgAta16
	cbd[1] = sgAtaProtoNonData
	cbd[2] = sgCdb2CheckCond
	cbd[13] = ataUsingLba
	cbd[14] = command
	sendSgio(f, cbd)
}

func sendSgio(f *os.File, inqCmdBlk [sgAta16Len]uint8) {
	senseBuf := make([]byte, sgio.SENSE_BUF_LEN)
	ioHdr := &sgio.SgIoHdr{
		InterfaceID:    int32('S'),
		DxferDirection: SgDxferNone,
		Cmdp:           &inqCmdBlk[0],
		CmdLen:         sgAta16Len,
		Sbp:            &senseBuf[0],
		MxSbLen:        sgio.SENSE_BUF_LEN,
		Timeout:        0,
	}

	if err := sgio.SgioSyscall(f, ioHdr); err != nil {
		log.Fatalln(err)
	}

	if err := sgio.CheckSense(ioHdr, &senseBuf); err != nil {
		log.Fatalln(err)
	}
}

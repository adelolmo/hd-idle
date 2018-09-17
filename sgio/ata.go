package sgio

import (
	"github.com/benmcclelland/sgio"
	"log"
	"os"
)

const (
	SG_ATA_16     = 0x85
	SG_ATA_16_LEN = 16

	SG_ATA_PROTO_NON_DATA = 3 << 1
	SG_CDB2_CHECK_COND    = 1 << 5
	ATA_USING_LBA         = 1 << 6

	ATA_OP_STANDBYNOW1 = 0xe0 // https://wiki.osdev.org/ATA/ATAPI_Power_Management
	ATA_OP_STANDBYNOW2 = 0x94 // Retired in ATA4. Did not coexist with ATAPI.
)

func StopAtaDevice(device string) {
	f, err := openDevice(device)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	sendAtaCommand(f, ATA_OP_STANDBYNOW1)
	sendAtaCommand(f, ATA_OP_STANDBYNOW2)
}

func sendAtaCommand(f *os.File, command uint8) {
	var cbd [SG_ATA_16_LEN]uint8
	cbd[0] = SG_ATA_16
	cbd[1] = SG_ATA_PROTO_NON_DATA
	cbd[2] = SG_CDB2_CHECK_COND
	cbd[13] = ATA_USING_LBA
	cbd[14] = command
	sendSgio(f, cbd)
}

func sendSgio(f *os.File, inqCmdBlk [SG_ATA_16_LEN]uint8) {
	senseBuf := make([]byte, sgio.SENSE_BUF_LEN)
	ioHdr := &sgio.SgIoHdr{
		InterfaceID:    int32('S'),
		DxferDirection: SG_DXFER_NONE,
		Cmdp:           &inqCmdBlk[0],
		CmdLen:         SG_ATA_16_LEN,
		Sbp:            &senseBuf[0],
		MxSbLen:        sgio.SENSE_BUF_LEN,
		Timeout:        0,
	}

	err := sgio.SgioSyscall(f, ioHdr)
	if err != nil {
		log.Fatalln(err)
	}

	err = sgio.CheckSense(ioHdr, &senseBuf)
	if err != nil {
		log.Fatalln(err)
	}
}

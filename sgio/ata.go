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
)

const (
	sgAta16 = 0x85 // ATA PASS-THROUGH(16)
	sgAta12 = 0xa1 // ATA PASS-THROUGH (12)

	sgAtaProtoNonData = 3 << 1
	ataUsingLba       = 1 << 6

	ataOpDoorUnlock = 0xdf

	ataOpStandbyNow1 = 0xe0 // https://wiki.osdev.org/ATA/ATAPI_Power_Management
	ataOpStandbyNow2 = 0x94 // Retired in ATA4. Did not coexist with ATAPI.
)

func StopAtaDevice(device string, preferAta12 bool) error {
	f, err := openDevice(device)
	if err != nil {
		return err
	}

	if preferAta12 {
		if err = issueAta12Standby(err, f); err != nil {
			return err
		}

	} else {
		if err = sendAtaCommand(f, ataOpStandbyNow1); err != nil {
			if err = sendAtaCommand(f, ataOpStandbyNow2); err != nil {
				return err
			}
		}
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("cannot close file %s. Error: %s", device, err)
	}
	return nil
}

func issueAta12Standby(err error, f *os.File) error {
	cbd := ataDoorUnlock()
	if err = sendSgio(f, cbd); err != nil {
		return err
	}
	if err = sendSgio(f, ata12Standby()); err != nil {
		return err
	}
	return nil
}

func ataDoorUnlock() []uint8 {
	cbd := []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0} // len 12
	cbd[0] = ataOpDoorUnlock
	cbd[1] = 0x10
	cbd[4] = 0x01
	cbd[6] = 0x72
	cbd[7] = 0x0f
	cbd[11] = 0xfd
	return cbd
}

func ata12Standby() []uint8 {
	fmt.Println(" issuing standby command")
	cbd := []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0} // len 12
	cbd[0] = ataOpDoorUnlock
	cbd[1] = 0x10
	cbd[10] = 0xa0 // device port
	cbd[11] = ataOpStandbyNow1
	return cbd
}

func sendAtaCommand(f *os.File, command uint8) error {

	cbd := []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0} // len 16
	cbd[0] = sgAta16
	cbd[1] = sgAtaProtoNonData
	cbd[13] = ataUsingLba
	cbd[14] = command
	return sendSgio(f, cbd)
}

func sendSgio(f *os.File, inqCmdBlk []uint8) error {
	senseBuf := make([]byte, sgio.SENSE_BUF_LEN)
	ioHdr := &sgio.SgIoHdr{
		InterfaceID:    'S',                   //  0	4
		DxferDirection: SgDxferNone,           //  4 	4
		CmdLen:         uint8(len(inqCmdBlk)), //  8	1
		MxSbLen:        sgio.SENSE_BUF_LEN,    //  9	1
		Cmdp:           &inqCmdBlk[0],         // 24	8
		Sbp:            &senseBuf[0],          // 32	8
		Timeout:        0,                     // 40	4
	}

	dumpBytes(inqCmdBlk)

	if err := sgio.SgioSyscall(f, ioHdr); err != nil {
		return err
	}

	if err := sgio.CheckSense(ioHdr, &senseBuf); err != nil {
		return err
	}
	return nil
}

func dumpBytes(p []uint8) {
	fmt.Print("outgoing cdb:  ")
	for i := range p {
		fmt.Printf("%02x ", p[i])
	}
	fmt.Print("\n")
}

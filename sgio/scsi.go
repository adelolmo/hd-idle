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
)

// https://en.wikipedia.org/wiki/SCSI_command
const startStopUnit = 0x1b

func StartStopScsiDevice(device string, powerCondition uint8) error {
	f, err := openDevice(device)
	if err != nil {
		return err
	}

	senseBuf := make([]byte, sgio.SENSE_BUF_LEN)
	//See https://www.seagate.com/files/staticfiles/support/docs/manual/Interface%20manuals/100293068j.pdf - 3.49 START STOP UNIT command
	inqCmdBlk := []uint8{
		startStopUnit,
		0,                   //Reserved (7 bit) + IMMED
		0,                   //Reserved (8 bit)
		0,                   //Reserved (4 bit) + POWER CONDITION MODIFER
		powerCondition << 4, //POWER CONDITION + Reserved (1 bit) + NO_ FLUSH + LOEJ + LOEJ
		0}                   //CONTROL
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
	return nil
}

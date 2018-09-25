package sgio

import (
	"bytes"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	SG_GET_VERSION_NUM   = 0x2282
	SG_IO                = 0x2285
	SG_INFO_OK_MASK      = 0x1
	SG_INFO_OK           = 0x0
	SG_DXFER_TO_DEV      = -2
	SG_DXFER_FROM_DEV    = -3
	SG_DXFER_TO_FROM_DEV = -4
	INQ_CMD_CODE         = 0x12
	INQ_REPLY_LEN        = 96
	SENSE_BUF_LEN        = 32
	TIMEOUT_20_SECS      = 20000
)

// pahole for sg_io_hdr_t on amd64
/*
 * struct sg_io_hdr {
 * 	int                        interface_id;         //     0     4
 * 	int                        dxfer_direction;      //     4     4
 * 	unsigned char              cmd_len;              //     8     1
 * 	unsigned char              mx_sb_len;            //     9     1
 * 	short unsigned int         iovec_count;          //    10     2
 * 	unsigned int               dxfer_len;            //    12     4
 * 	void *                     dxferp;               //    16     8
 * 	unsigned char *            cmdp;                 //    24     8
 * 	unsigned char *            sbp;                  //    32     8
 * 	unsigned int               timeout;              //    40     4
 * 	unsigned int               flags;                //    44     4
 * 	int                        pack_id;              //    48     4
 *
 * 	// XXX 4 bytes hole, try to pack
 *
 * 	void *                     usr_ptr;              //    56     8
 * 	// --- cacheline 1 boundary (64 bytes) ---
 * 	unsigned char              status;               //    64     1
 * 	unsigned char              masked_status;        //    65     1
 * 	unsigned char              msg_status;           //    66     1
 * 	unsigned char              sb_len_wr;            //    67     1
 * 	short unsigned int         host_status;          //    68     2
 * 	short unsigned int         driver_status;        //    70     2
 * 	int                        resid;                //    72     4
 * 	unsigned int               duration;             //    76     4
 * 	unsigned int               info;                 //    80     4
 *
 * 	// size: 88, cachelines: 2, members: 22
 * 	// sum members: 80, holes: 1, sum holes: 4
 * 	// padding: 4
 * 	// last cacheline: 24 bytes
 * };
 */

// SgIoHdr is our version of sg_io_hdr_t that gets passed to the SG_IO ioctl
type SgIoHdr struct {
	InterfaceID    int32
	DxferDirection int32
	CmdLen         uint8
	MxSbLen        uint8
	IovecCount     uint16
	DxferLen       uint32
	Dxferp         *byte
	Cmdp           *uint8
	Sbp            *byte
	Timeout        uint32
	Flags          uint32
	PackID         int32
	pad0           [4]byte
	UsrPtr         *byte
	Status         uint8
	MaskedStatus   uint8
	MsgStatus      uint8
	SbLenWr        uint8
	HostStatus     uint16
	DriverStatus   uint16
	Resid          int32
	Duration       uint32
	Info           uint32
}

func TestUnitReady(f *os.File) error {
	senseBuf := make([]byte, SENSE_BUF_LEN)
	inqCmdBlk := []uint8{0, 0, 0, 0, 0, 0}
	ioHdr := &SgIoHdr{
		InterfaceID:    int32('S'),
		CmdLen:         uint8(len(inqCmdBlk)),
		MxSbLen:        SENSE_BUF_LEN,
		DxferDirection: SG_DXFER_FROM_DEV,
		Cmdp:           &inqCmdBlk[0],
		Sbp:            &senseBuf[0],
		Timeout:        TIMEOUT_20_SECS,
	}

	err := SgioSyscall(f, ioHdr)
	if err != nil {
		return err
	}

	err = CheckSense(ioHdr, &senseBuf)
	if err != nil {
		return err
	}

	return nil
}

func CheckSense(i *SgIoHdr, s *[]byte) error {
	var b bytes.Buffer
	if (i.Info & SG_INFO_OK_MASK) != SG_INFO_OK {
		_, err := b.WriteString(
			fmt.Sprintf("SCSI response not ok\n"+
				"SCSI status: %v host status: %v driver status: %v",
				i.Status, i.HostStatus, i.DriverStatus))
		if err != nil {
			return err
		}
		if i.SbLenWr > 0 {
			_, err := b.WriteString(
				fmt.Sprintf("\nSENSE:\n%v\n%v",
					dumpHex(*s), GetErrString((*s)[12], (*s)[13])))
			if err != nil {
				return err
			}
		}
		return fmt.Errorf(b.String())
	}
	return nil
}

func SgioSyscall(f *os.File, i *SgIoHdr) error {
	return ioctl(f.Fd(), SG_IO, uintptr(unsafe.Pointer(i)))
}

func ioctl(fd, cmd, ptr uintptr) error {
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, cmd, ptr)
	if err != 0 {
		return err
	}
	return nil
}

func OpenScsiDevice(fname string) (*os.File, error) {
	f, err := os.OpenFile(fname, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	var version uint32
	if (ioctl(f.Fd(), SG_GET_VERSION_NUM, uintptr(unsafe.Pointer(&version))) != nil) || (version < 30000) {
		return nil, fmt.Errorf("device does not appear to be an sg device")
	}
	return f, nil
}

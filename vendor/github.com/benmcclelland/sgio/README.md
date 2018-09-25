# sgio
golang library for issuing SCSI commands with SG_IO ioctl

[![godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/benmcclelland/sgio)

See TestUnitReady() for example function using SG_IO

example:
```
f, err := OpenScsiDevice("/dev/sg0")
if err != nil {
	log.Fatalln(err)
}
defer f.Close()
```
Fill out SgIoHdr for SCSI command
```
ioHdr := &SgIoHdr{...}
err := SgioSyscall(f, ioHdr)
if err != nil {
	log.Fatalln(err)
}

err = CheckSense(ioHdr, &senseBuf)
if err != nil {
	log.Fatalln(err)
}
```
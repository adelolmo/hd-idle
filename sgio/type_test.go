package sgio

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	tmpDir = "/tmp/hd-idle/ata"
)

func TestAtaDevice_deviceType(t *testing.T) {
	type fields struct {
		device                         string
		debug                          bool
		fsRoot                         string
		idVendor, idProduct, bcdDevice string
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "find jmicron controller",
			fields: fields{
				device:    "/dev/sde",
				debug:     true,
				fsRoot:    filepath.Join(tmpDir, "sys", "block"),
				idVendor:  "152d",
				idProduct: "2339",
				bcdDevice: "100",
			},
			want: Jmicron,
		},
		{
			name: "unknown device",
			fields: fields{
				device:    "/dev/sde",
				debug:     true,
				fsRoot:    filepath.Join(tmpDir, "sys", "block"),
				idVendor:  "1058",
				idProduct: "25a3",
				bcdDevice: "1021",
			},
			want: Unknown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ad := AtaDevice{
				device: tt.fields.device,
				debug:  tt.fields.debug,
				fsRoot: tt.fields.fsRoot,
			}

			err := os.RemoveAll(tmpDir)
			if err != nil {
				log.Fatal(err)
			}
			infoDir := filepath.Join(tmpDir, "/sys/devices/pci0000:00/0000:00:15.0/usb2/2-2/2-2.3/2-2.3.2")
			diskname := strings.Split(tt.fields.device, "/")[2]
			deviceRoot := infoDir + "/2-2.3.2:1.0/host5/target5:0:0/5:0:0:0/block/" + diskname
			_ = os.MkdirAll(deviceRoot, 0755)
			_ = os.WriteFile(filepath.Join(infoDir, "idVendor"), []byte(tt.fields.idVendor), 0666)
			_ = os.WriteFile(filepath.Join(infoDir, "idProduct"), []byte(tt.fields.idProduct), 0666)
			_ = os.WriteFile(filepath.Join(infoDir, "bcdDevice"), []byte(tt.fields.bcdDevice), 0666)
			_ = os.MkdirAll(filepath.Join(tmpDir, "/sys/block"), 0755)
			if err = os.Symlink(deviceRoot, filepath.Join(tmpDir, "/sys/block", diskname)); err != nil {
				log.Fatal(err)
			}
			if got := ad.deviceType(); got != tt.want {
				t.Errorf("deviceType() = %v, want %v", got, tt.want)
			}
		})
	}
}

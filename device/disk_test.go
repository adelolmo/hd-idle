package device

import (
	"fmt"
	"os"
	"testing"
)

func TestRealPath(t *testing.T) {
	err := os.RemoveAll("/tmp/dev")
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll("/tmp/dev/disk/by-id", os.ModePerm)
	if err != nil {
		panic("cannot create tmp dir")
	}
	type args struct {
		path string
	}
	tests := []struct {
		name        string
		args        args
		want        string
		withSymlink bool
	}{
		{
			name:        "only device name",
			args:        args{path: "sda"},
			want:        "sda",
			withSymlink: false,
		},
		{
			name:        "full device path",
			args:        args{path: "/tmp/dev/sda"},
			want:        "sda",
			withSymlink: false,
		},
		{
			name:        "symlink by id",
			args:        args{path: "/tmp/dev/disk/by-id/ata-SAMSUNG_HD103SJ"},
			want:        "sdc",
			withSymlink: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			disk := fmt.Sprintf("/tmp/dev/%s", tt.want)
			_, err := os.Create(disk)
			if err != nil {
				panic(err)
			}
			if tt.withSymlink {
				err = os.Symlink(disk, tt.args.path)
				if err != nil {
					panic(err)
				}
			}
			if got := RealPath(tt.args.path); got != tt.want {
				t.Errorf("RealPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

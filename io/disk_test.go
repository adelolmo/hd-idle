package io

import (
	"fmt"
	"os"
	"testing"
)

func TestRealPath(t *testing.T) {

	type args struct {
		path string
	}
	tests := []struct {
		name          string
		args          args
		want          string
		symlinkTarget string
		expectError   bool
	}{
		{
			name: "only device name",
			args: args{path: "sda"},
			want: "sda",
		},
		{
			name: "full device path",
			args: args{path: "/tmp/dev/sda"},
			want: "sda",
		},
		{
			name:        "wrong symlink by id",
			args:        args{path: "/tmp/dev/disk/by-id/ata-SAMSUNG_HD103SJ"},
			want:        "",
			expectError: true,
		},
		{
			name:          "symlink by id",
			args:          args{path: "/tmp/dev/disk/by-id/ata-SAMSUNG_HD103SJ"},
			want:          "sdc",
			symlinkTarget: "/tmp/dev/sdc",
		},
	}
	for _, tt := range tests {
		err := os.RemoveAll("/tmp/dev")
		if err != nil {
			panic(err)
		}
		err = os.MkdirAll("/tmp/dev/disk/by-id", os.ModePerm)
		if err != nil {
			panic("cannot create tmp dir")
		}
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.want) > 0 {
				disk := fmt.Sprintf("/tmp/dev/%s", tt.want)
				_, err := os.Create(disk)
				if err != nil {
					panic(err)
				}
				if len(tt.symlinkTarget) > 0 {
					err = os.Symlink(tt.symlinkTarget, tt.args.path)
					if err != nil {
						panic(err)
					}
				}
			}
			got, err := RealPath(tt.args.path)

			if err != nil && tt.expectError == false {
				panic(err)
			}
			if got != tt.want {
				t.Errorf("RealPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

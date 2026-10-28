package main

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestResolveSymlinks(t *testing.T) {
	const testDir = "/tmp/hd-idle-test"
	const byIdDir = testDir + "/dev/disk/by-id"
	const givenName = byIdDir + "/test-disk"
	const logFile = testDir + "/hd-idle.log"

	tests := []struct {
		name          string
		symlinkPolicy int
		deviceName    string
		targets       []string
		wantName      string
		wantLogLines  int
	}{
		{
			name:          "unresolved device resolves on next cycle",
			symlinkPolicy: symlinkResolveRetry,
			deviceName:    "",
			targets:       []string{"sda"},
			wantName:      "sda",
			wantLogLines:  1,
		},
		{
			name:          "resolved device with same target stays unchanged",
			symlinkPolicy: symlinkResolveRetry,
			deviceName:    "sda",
			targets:       []string{"sda"},
			wantName:      "sda",
			wantLogLines:  0,
		},
		{
			name:          "resolved device picks up changed symlink target",
			symlinkPolicy: symlinkResolveRetry,
			deviceName:    "sda",
			targets:       []string{"sda", "sdb"},
			wantName:      "sdb",
			wantLogLines:  1,
		},
		{
			name:          "failed resolution keeps previous device name",
			symlinkPolicy: symlinkResolveRetry,
			deviceName:    "sda",
			targets:       []string{""},
			wantName:      "sda",
			wantLogLines:  0,
		},
		{
			name:          "symlink policy 0 does not resolve",
			symlinkPolicy: symlinkResolveOnce,
			deviceName:    "",
			targets:       []string{"sda"},
			wantName:      "",
			wantLogLines:  0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.RemoveAll(testDir); err != nil {
				t.Fatalf("cannot remove test dir: %v", err)
			}
			if err := os.MkdirAll(byIdDir, os.ModePerm); err != nil {
				t.Fatalf("cannot create test dir: %v", err)
			}
			config := &Config{
				Defaults: DefaultConf{
					SymlinkPolicy: tt.symlinkPolicy,
					LogFile:       logFile,
				},
				Devices: []DeviceConf{{
					Name:      tt.deviceName,
					GivenName: givenName,
				}},
				NameMap: map[string]string{tt.deviceName: givenName},
			}
			for _, target := range tt.targets {
				if err := os.Remove(givenName); err != nil && !os.IsNotExist(err) {
					t.Fatalf("cannot remove stale symlink: %v", err)
				}
				if target != "" {
					disk := testDir + "/dev/" + target
					if err := os.WriteFile(disk, nil, 0666); err != nil {
						t.Fatalf("cannot create device file: %v", err)
					}
					if err := os.Symlink(disk, givenName); err != nil {
						t.Fatalf("cannot create symlink: %v", err)
					}
				}
				resolveSymlinks(config)
			}
			device := config.Devices[0]
			if device.Name != tt.wantName {
				t.Errorf("device name = %q, want %q", device.Name, tt.wantName)
			}
			wantNameMap := map[string]string{tt.wantName: givenName}
			if !reflect.DeepEqual(config.NameMap, wantNameMap) {
				t.Errorf("NameMap = %v, want %v", config.NameMap, wantNameMap)
			}
			content, readErr := os.ReadFile(logFile)
			if readErr != nil && tt.wantLogLines > 0 {
				t.Fatalf("cannot read log file: %v", readErr)
			}
			if got := strings.Count(string(content), "\n"); got != tt.wantLogLines {
				t.Errorf("log file has %d entries, want %d: %q", got, tt.wantLogLines, content)
			}
		})
	}
}

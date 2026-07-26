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

package main

import (
	"errors"
	"testing"
	"time"
)

func TestSpindownResultDeterminesSpunDownState(t *testing.T) {
	tests := []struct {
		name              string
		spindownErr       error
		wantSpunDown      bool
		wantSpinDownAtSet bool
	}{
		{
			name:              "successful spindown marks the disk as spun down",
			spindownErr:       nil,
			wantSpunDown:      true,
			wantSpinDownAtSet: true,
		},
		{
			name:              "failed spindown leaves the disk as still spinning",
			spindownErr:       errors.New("cannot spindown scsi disk /dev/sda"),
			wantSpunDown:      false,
			wantSpinDownAtSet: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			now = time.Now()
			lastNow = now
			previousSnapshots = []DiskStats{idleDiskStats(now)}

			updateState(DiskStats{Name: "sda", Reads: 100, Writes: 100}, idleDiskConfig(),
				func(device, command string, powerCondition uint8, debug bool) error {
					return test.spindownErr
				})

			if previousSnapshots[0].SpunDown != test.wantSpunDown {
				t.Errorf("Expected SpunDown %v but found %v",
					test.wantSpunDown, previousSnapshots[0].SpunDown)
			}
			if !previousSnapshots[0].SpinDownAt.IsZero() != test.wantSpinDownAtSet {
				t.Errorf("Expected SpinDownAt set %v but found %v",
					test.wantSpinDownAtSet, !previousSnapshots[0].SpinDownAt.IsZero())
			}
		})
	}
}

func TestFailedSpindownIsRetriedOncePerIdlePeriod(t *testing.T) {
	now = time.Now()
	lastNow = now
	previousSnapshots = []DiskStats{idleDiskStats(now)}

	attempts := 0
	failing := func(device, command string, powerCondition uint8, debug bool) error {
		attempts++
		return errors.New("cannot spindown scsi disk /dev/sda")
	}
	snapshot := DiskStats{Name: "sda", Reads: 100, Writes: 100}
	config := idleDiskConfig()

	updateState(snapshot, config, failing)
	if attempts != 1 {
		t.Fatalf("Expected 1 attempt but found %d", attempts)
	}

	now = now.Add(5 * time.Second)
	lastNow = now
	updateState(snapshot, config, failing)
	if attempts != 1 {
		t.Fatalf("Expected no retry within the idle period but found %d attempts", attempts)
	}

	now = now.Add(11 * time.Second)
	lastNow = now
	updateState(snapshot, config, failing)
	if attempts != 2 {
		t.Fatalf("Expected 2 attempts after a further idle period but found %d", attempts)
	}
}

func idleDiskConfig() *Config {
	return &Config{
		Defaults: DefaultConf{CommandType: SCSI},
		SkewTime: time.Minute,
		NameMap:  map[string]string{},
	}
}

func idleDiskStats(at time.Time) DiskStats {
	return DiskStats{
		Name:        "sda",
		GivenName:   "sda",
		IdleTime:    10 * time.Second,
		CommandType: SCSI,
		Reads:       100,
		Writes:      100,
		LastIoAt:    at.Add(-time.Hour),
		SpinUpAt:    at.Add(-time.Hour),
	}
}

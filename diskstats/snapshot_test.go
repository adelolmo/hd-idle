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

package diskstats

import (
	"strings"
	"testing"
)

func TestTakeSnapshot(t *testing.T) {
	s := `   7       0 loop0 0 0 0 0 0 0 0 0 0 0 0
   7       1 loop1 0 0 0 0 0 0 0 0 0 0 0
   7       2 loop2 0 0 0 0 0 0 0 0 0 0 0
   7       3 loop3 0 0 0 0 0 0 0 0 0 0 0
   7       4 loop4 0 0 0 0 0 0 0 0 0 0 0
   7       5 loop5 0 0 0 0 0 0 0 0 0 0 0
   7       6 loop6 0 0 0 0 0 0 0 0 0 0 0
   7       7 loop7 0 0 0 0 0 0 0 0 0 0 0
 179       0 mmcblk0 133145 53235 6878634 3020910 1544414 1254150 48441345 240124500 0 13439150 243142800
 179       1 mmcblk0p1 80 40 1760 210 1 0 1 0 0 140 210
 179       2 mmcblk0p2 132931 53195 6874482 3020440 1544413 1254150 48441344 240124500 0 13439000 243278260
   8       0 sda 321553 158156 37537568 5961590 50820 94361 10439592 26691430 0 3357150 32650910
   8       1 sda1 321454 158156 37536344 5725790 50820 94361 10439592 26691430 0 3121370 32415240
   8      32 sdc 52147 2738 6494584 913050 28092 1251 6370936 8938800 0 506360 9852970
   8      33 sdc1 52087 2738 6493672 905390 28092 1251 6370936 8938800 0 498700 9892750
   8      16 sdb 5650742 34516 727476416 92732820 1728864 35618 404215912 705303450 0 22944140 798112260
   8      17 sdb1 5650643 34516 727475192 92673920 1728864 35618 404215912 705303450 0 22893010 798071230`

	stats := ReadSnapshot(strings.NewReader(s))

	expected := []DiskStats{
		{Name: "sda", Reads: 37537568, Writes: 10439592},
		{Name: "sdc", Reads: 6494584, Writes: 6370936},
		{Name: "sdb", Reads: 727476416, Writes: 404215912},
	}

	if len(expected) != len(stats) {
		t.Fatalf("Expected %d disks but found %d", len(expected), len(stats))
	}
	for i := 0; i < len(expected); i++ {
		exp := expected[i]
		act := stats[i]

		if exp != act {
			t.Fatalf("Expected %v but found %v", exp, act)
		}
	}
}
